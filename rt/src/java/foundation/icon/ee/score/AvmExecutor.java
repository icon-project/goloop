/*
 * Copyright 2019 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.ee.score;

import foundation.icon.ee.types.Address;
import i.IInstrumentation;
import i.IInstrumentationFactory;
import i.InstrumentationHelpers;
import i.RuntimeAssertionError;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.core.DAppCreator;
import org.aion.avm.core.DAppExecutor;
import org.aion.avm.core.DAppLoader;
import org.aion.avm.core.IExternalState;
import org.aion.avm.core.ReentrantDAppStack;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.avm.core.util.TransactionResultUtil;
import org.aion.kernel.AvmWrappedTransactionResult;
import org.aion.kernel.AvmWrappedTransactionResult.AvmInternalError;
import org.aion.parallel.TransactionTask;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;
import org.aion.types.TransactionResult;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.math.BigInteger;

public class AvmExecutor {
    private static final Logger logger = LoggerFactory.getLogger(AvmExecutor.class);

    private final IInstrumentationFactory instrumentationFactory;
    private final boolean preserveDebuggability;
    private final boolean enableVerboseContractErrors;
    private final boolean enableBlockchainPrintln;
    private Loader loader;
    private IInstrumentation instrumentation;
    private TransactionTask task;

    public AvmExecutor(IInstrumentationFactory factory, AvmConfiguration config, Loader loader) {
        this.instrumentationFactory = factory;
        this.preserveDebuggability = config.preserveDebuggability;
        this.enableVerboseContractErrors = config.enableVerboseContractErrors;
        this.enableBlockchainPrintln = config.enableBlockchainPrintln;
        this.loader = loader;
    }

    public void start() {
        instrumentation = instrumentationFactory.createInstrumentation();
        InstrumentationHelpers.attachThread(instrumentation);
    }

    public TransactionResult run(IExternalState kernel, Transaction transaction, Address origin) {
        if (task==null) {
            return runExternal(kernel, transaction, origin).unwrap();
        } else {
            return runInternal(kernel, transaction, origin).unwrap();
        }
    }

    private AvmWrappedTransactionResult runExternal(IExternalState kernel, Transaction transaction, Address origin) {
        // Get the first task
        task = new TransactionTask(kernel, transaction, 0,
                                                           origin != null ? new AionAddress(origin) : null);

        // Attach the IInstrumentation helper to the task to support asynchronous abort
        // Instrumentation helper will abort the execution of the transaction by throwing an exception during chargeEnergy call
        // Aborted transaction will be retried later
        task.startNewTransaction();
        task.attachInstrumentationForThread();
        AvmWrappedTransactionResult result = processTransaction();
        task.detachInstrumentationForThread();

        if (result.isAborted()) {
            // If this was an abort, we want to clear the abort state on the instrumentation for this thread, since
            // this is the point where that is "handled".
            // Note that this is safe to do here since the instrumentation isn't exposed to any other threads.
            instrumentation.clearAbortState();
            logger.trace("Abort " + task.getIndex());
        }
        logger.trace("{}", result);
        task = null;
        return result;
    }

    private AvmWrappedTransactionResult runInternal(IExternalState kernel, Transaction transaction, Address origin) {
        return runCommon(kernel, transaction);
    }

    private AvmWrappedTransactionResult processTransaction() {
        AvmInternalError error = AvmInternalError.NONE;
        Transaction tx = task.getTransaction();
        RuntimeAssertionError.assertTrue(tx != null);

        BigInteger value = tx.value;
        if (value.compareTo(BigInteger.ZERO) < 0) {
            error = AvmInternalError.REJECTED_INVALID_VALUE;
        }

        if (tx.isCreate) {
            if (!task.getThisTransactionalKernel().isValidEnergyLimitForCreate(tx.energyLimit)) {
                error = AvmInternalError.REJECTED_INVALID_ENERGY_LIMIT;
            }
        } else {
            if (!task.getThisTransactionalKernel().isValidEnergyLimitForNonCreate(tx.energyLimit)) {
                error = AvmInternalError.REJECTED_INVALID_ENERGY_LIMIT;
            }
        }
        // exit if validation check fails
        if (error != AvmInternalError.NONE) {
            return TransactionResultUtil.newRejectedResultWithEnergyUsed(error, tx.energyLimit);
        }
        return runCommon(task.getThisTransactionalKernel(), tx);
    }

    private AvmWrappedTransactionResult runCommon(IExternalState kernel, Transaction tx) {
        /*
         * Run the common logic with the parent kernel as the top-level one.
         * After this point, no rejection should occur.
         */
        // start with the successful result
        AvmWrappedTransactionResult result = TransactionResultUtil.newSuccessfulResultWithEnergyUsed(0);

        AionAddress senderAddress = tx.senderAddress;
        AionAddress recipient = tx.destinationAddress;

        if (tx.isCreate) {
            logger.trace("=== DAppCreator ===");
            result = DAppCreator.create(kernel, task,
                    senderAddress, recipient, tx, result,
                    this.preserveDebuggability, this.enableVerboseContractErrors, this.enableBlockchainPrintln);
        } else {
            LoadedDApp dapp;

            // See if this call is trying to reenter one already on this call-stack.  If so, we will need to partially resume its state.
            ReentrantDAppStack.ReentrantState stateToResume = task.getReentrantDAppStack().tryShareState(recipient);

            if ((null != stateToResume)) {
                dapp = stateToResume.dApp;
                // Call directly and don't interact with DApp cache (we are reentering the state, not the origin of it).
                logger.trace("=== DAppExecutor === call 1");
                result = DAppExecutor.call(kernel, dapp, stateToResume, task,
                        senderAddress, recipient, tx, result,
                        this.enableVerboseContractErrors, true, this.enableBlockchainPrintln);
            } else {
                try {
                    dapp = loader.load(recipient, kernel, preserveDebuggability);
                } catch (IOException e) {
                    throw RuntimeAssertionError.unexpected(e);
                }
                logger.trace("=== DAppExecutor === call 2");
                result = DAppExecutor.call(kernel, dapp, null, task,
                        senderAddress, recipient, tx, result,
                        this.enableVerboseContractErrors, false, this.enableBlockchainPrintln);
            }
        }

        if (result.isSuccess()) {
            kernel.commit();
        } else if (result.isFailedUnexpected()) {
            logger.error("Unexpected error during transaction execution!", result.exception);
        }

        if (!result.isAborted()){
            result = TransactionResultUtil.setExternalState(result, task.getThisTransactionalKernel());
            task.outputFlush();
        }
        return result;
    }

    public void shutdown() {
        InstrumentationHelpers.detachThread(instrumentation);
        instrumentationFactory.destroyInstrumentation(instrumentation);
    }
}
