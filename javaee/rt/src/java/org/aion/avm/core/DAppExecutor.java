package org.aion.avm.core;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.DAppRuntimeState;
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Status;
import foundation.icon.ee.types.Transaction;
import i.AvmError;
import i.AvmException;
import i.GenericPredefinedException;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.InstrumentationHelpers;
import i.InternedClasses;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.parallel.TransactionTask;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class DAppExecutor {
    private static final Logger logger = LoggerFactory.getLogger(DAppExecutor.class);

    public static Result call(IExternalState externalState,
                              LoadedDApp dapp,
                              ReentrantDAppStack.ReentrantState stateToResume,
                              TransactionTask task,
                              Address senderAddress,
                              Address dappAddress,
                              Transaction tx,
                              AvmConfiguration conf) throws AvmError {
        Result result = null;

        // Note that the instrumentation is just a per-thread access to the state stack - we can grab it at any time as it never changes for this thread.
        IInstrumentation threadInstrumentation = IInstrumentation.attachedThreadInstrumentation.get();
        
        // We need to get the interned classes before load the graph since it might need to instantiate class references.
        InternedClasses initialClassWrappers = dapp.getInternedClasses();

        var oldRS = task.getReentrantDAppStack().getRuntimeState(task.getPrevEID());
        if (oldRS == null) {
            oldRS = dapp.loadRuntimeState(externalState);
        } else {
            dapp.loadRuntimeState(oldRS);
        }
        var nextHashCode = oldRS.getGraph().getNextHash();

        // Used for deserialization billing
        int rawGraphDataLength = oldRS.getGraph().getGraphData().length;

        // Note that we need to store the state of this invocation on the reentrant stack in case there is another call into the same app.
        // This is required so that the call() mechanism can access it to save/reload its ContractEnvironmentState and so that the underlying
        // instance loader (ReentrantGraphProcessor/ReflectionStructureCodec) can be notified when it becomes active/inactive (since it needs
        // to know if it is loading an instance
        ReentrantDAppStack.ReentrantState thisState = new ReentrantDAppStack.ReentrantState(dappAddress, dapp);
        var prevState = task.getReentrantDAppStack().getTop();
        task.getReentrantDAppStack().pushState(thisState);

        IBlockchainRuntime br = new BlockchainRuntimeImpl(externalState,
                                                          task,
                                                          senderAddress,
                                                          dappAddress,
                                                          tx,
                                                          dapp.runtimeSetup,
                                                          dapp,
                                                          conf.enableContextPrintln);
        FrameContextImpl fc = new FrameContextImpl(externalState);
        InstrumentationHelpers.pushNewStackFrame(dapp.runtimeSetup, dapp.loader, tx.getLimit(), nextHashCode, initialClassWrappers, fc);
        IBlockchainRuntime previousRuntime = dapp.attachBlockchainRuntime(br);

        try {
            // It is now safe for us to bill for the cost of loading the graph (the cost is the same, whether this came from the caller or the disk).
            // (note that we do this under the try since aborts can happen here)
            // Do not charge defaultGet as defaultGet is considered in default
            // CALL value.
            threadInstrumentation.chargeEnergy(
                    externalState.getStepCost().get() * rawGraphDataLength
            );

            // Call the main within the DApp.
            Object ret;
            try {
                ret = dapp.callMethod(tx.getMethod(), tx.getParams());
            } finally {
                externalState.waitForCallbacks();
            }

            var newRS = dapp.saveRuntimeState();

            if (externalState.isReadOnly() && !oldRS.isAcceptableChangeInReadOnly(newRS)) {
                throw new GenericPredefinedException(Status.AccessDenied);
            }

            if (newRS.getGraph().equalGraphData(oldRS.getGraph())) {
                newRS = new DAppRuntimeState(newRS, oldRS.getGraph().getNextHash());
            } else {
                var postOG = newRS.getGraph();
                byte[] postCallGraphData = postOG.getGraphData();
                var effectiveLen = Math.max(externalState.getStepCost().replaceBase(),
                        postCallGraphData.length);
                threadInstrumentation.chargeEnergy(
                        effectiveLen * externalState.getStepCost().replace());
                if (null == stateToResume) {
                    // Save back the state before we return.
                    externalState.putObjectGraph(postOG);
                }
            }

            long energyUsed = tx.getLimit() - threadInstrumentation.energyLeft();
            result = new Result(Status.Success, energyUsed, ret);
            if (prevState != null) {
                prevState.inherit(thisState);
                prevState.setRuntimeState(task.getEID(), newRS, dappAddress);
            }
        } catch (AvmException e) {
            logger.trace("DApp invocation failed: {}", e.getMessage());
            if (conf.enableVerboseContractErrors) {
                e.printStackTrace();
            }
            long stepUsed = tx.getLimit() - threadInstrumentation.energyLeft();
            result = new Result(e.getCode(), stepUsed, e.getResultMessage());
        } finally {
            // Once we are done running this, no matter how it ended, we want to detach our thread from the DApp.
            InstrumentationHelpers.popExistingStackFrame(dapp.runtimeSetup);
            // This state was only here while we were running, in case someone else needed to change it so now we can pop it.
            task.getReentrantDAppStack().popState();
            // Re-attach the previously detached IBlockchainRuntime instance.
            dapp.attachBlockchainRuntime(previousRuntime);
        }
        return result;
    }
}
