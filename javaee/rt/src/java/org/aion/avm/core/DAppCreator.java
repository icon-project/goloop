package org.aion.avm.core;

import foundation.icon.ee.score.Transformer;
import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.Transaction;
import foundation.icon.ee.util.LogMarker;
import i.AvmError;
import i.AvmException;
import i.AvmThrowable;
import i.GenericPredefinedException;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.IRuntimeSetup;
import i.InstrumentationHelpers;
import i.OutOfStackException;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.avm.core.types.TransformedDappModule;
import org.aion.parallel.TransactionTask;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;

public class DAppCreator {
    private static final Logger logger = LoggerFactory.getLogger(DAppCreator.class);

    public static Result create(IExternalState externalState,
                                TransactionTask task,
                                Address senderAddress,
                                Address dappAddress,
                                Transaction tx,
                                AvmConfiguration conf) throws AvmError {
        IRuntimeSetup runtimeSetup = null;
        IBlockchainRuntime previousRuntime = null;
        Result result;
        LoadedDApp dapp = null;
        int flag = 0;
        try {
            Transformer transformer = new Transformer(
                    externalState,
                    conf);
            transformer.transform();
            TransformedDappModule transformedDapp = transformer.getBootstrapModule();
            dapp = DAppLoader.fromTransformed(
                    transformedDapp,
                    transformer.getAPIsBytes(),
                    conf.preserveDebuggability);
            runtimeSetup = dapp.runtimeSetup;

            // We start the nextHashCode at 1.
            int nextHashCode = 1;
            var prevState = task.getReentrantDAppStack().getTop();
            // we pass a null re-entrant state since we haven't finished initializing yet - nobody can call into us.
            task.getReentrantDAppStack().pushState();
            var thisState = task.getReentrantDAppStack().getTop();
            IBlockchainRuntime br = new BlockchainRuntimeImpl(externalState,
                                                              task,
                                                              senderAddress,
                                                              dappAddress,
                                                              tx,
                                                              runtimeSetup,
                                                              dapp);
            FrameContextImpl fc = new FrameContextImpl(externalState, true);
            InstrumentationHelpers.pushNewStackFrame(runtimeSetup, dapp.loader, tx.getLimit(), nextHashCode, dapp.getInternedClasses(), fc);
            previousRuntime = dapp.attachBlockchainRuntime(br);

            externalState.setTransformedCode(transformer.getTransformedCodeBytes());

            // Force the classes in the dapp to initialize so that the <clinit> is run (since we already saved the version without).
            IInstrumentation threadInstrumentation = IInstrumentation.attachedThreadInstrumentation.get();
            result = runClinitAndCreateMainInstance(dapp, threadInstrumentation, externalState, tx);
            if (prevState != null) {
                prevState.inherit(thisState);
                var newRS = dapp.saveRuntimeState();
                prevState.setRuntimeState(task.getEID(), newRS, externalState.getContractID());
                task.getReentrantDAppStack().cacheDApp(dapp,
                        externalState.getContractID(),
                        externalState.getCodeID());
            }
        } catch (AvmException e) {
            logger.trace("DApp deployment failed: {}", e.getMessage());
            if (conf.testMode) {
                e.printStackTrace();
            }
            var bos = new ByteArrayOutputStream();
            e.printStackTrace(new PrintStream(bos));
            logger.trace(LogMarker.Trace, bos.toString());
            long stepUsed = (runtimeSetup != null) ?
                    (tx.getLimit() - IInstrumentation.getEnergyLeft()) : 0;
            result = new Result(e.getCode(), stepUsed, e.getResultMessage());
        } finally {
            // Once we are done running this, no matter how it ended, we want to detach our thread from the DApp.
            if (null != runtimeSetup) {
                flag = InstrumentationHelpers.popExistingStackFrame(runtimeSetup);
                task.getReentrantDAppStack().popState();
                dapp.attachBlockchainRuntime(previousRuntime);
            }
        }
        result = result.updateStatus(result.getStatus()|flag);
        return result;
    }

    /**
     * Initializes all the classes in the dapp by running their clinit code.
     *
     * This method handles the following exceptions and ensures that if any of them are thrown
     * that they will be represented by the returned result (any other exceptions thrown here will
     * not be handled):
     * {@link OutOfStackException}, and {@link GenericPredefinedException}.
     *
     * @param dapp The dapp to run.
     * @param threadInstrumentation The thread instrumentation.
     * @param externalState The state of the world.
     * @param tx The transaction.
     * @return the result of initializing and billing the sender.
     */
    private static Result runClinitAndCreateMainInstance(LoadedDApp dapp,
                                                         IInstrumentation threadInstrumentation,
                                                         IExternalState externalState,
                                                         Transaction tx) throws AvmThrowable {
        try {
            dapp.forceInitializeAllClasses();
            dapp.initMainInstance(tx.getParams());
        } finally {
            externalState.waitForCallbacks();
        }

        // Save back the state before we return.
        var og = dapp.saveRuntimeState().getGraph();
        byte[] rawGraphData = og.getGraphData();
        var effectiveLen = rawGraphData.length;
        threadInstrumentation.chargeEnergy(
                externalState.getStepCost().setStorageSet(effectiveLen)
        );
        externalState.putObjectGraph(og);

        long energyUsed = tx.getLimit() - threadInstrumentation.energyLeft();
        return new Result(Status.Success, energyUsed, null);
    }
}
