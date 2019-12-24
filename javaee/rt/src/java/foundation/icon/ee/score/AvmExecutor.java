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
import foundation.icon.ee.types.Result;
import foundation.icon.ee.types.Status;
import i.IInstrumentation;
import i.IInstrumentationFactory;
import i.InstrumentationHelpers;
import i.RuntimeAssertionError;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.core.DAppCreator;
import org.aion.avm.core.DAppExecutor;
import org.aion.avm.core.IExternalState;
import org.aion.avm.core.ReentrantDAppStack;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.parallel.TransactionTask;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;
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

    public Result run(IExternalState kernel, Transaction transaction, Address origin) {
        if (task == null) {
            return runExternal(kernel, transaction, origin);
        } else {
            return runInternal(kernel, transaction);
        }
    }

    private Result runExternal(IExternalState kernel, Transaction transaction, Address origin) {
        // Get the first task
        task = new TransactionTask(kernel, transaction, 0, origin != null ? new AionAddress(origin) : null);

        // Attach the IInstrumentation helper to the task to support asynchronous abort
        // Instrumentation helper will abort the execution of the transaction by throwing an exception during chargeEnergy call
        // Aborted transaction will be retried later
        task.startNewTransaction();
        task.attachInstrumentationForThread();
        Result result = processTransaction();
        task.detachInstrumentationForThread();

        logger.trace("{}", result);
        task = null;
        return result;
    }

    private Result runInternal(IExternalState kernel, Transaction transaction) {
        return runCommon(kernel, transaction);
    }

    private Result processTransaction() {
        Transaction tx = task.getTransaction();
        RuntimeAssertionError.assertTrue(tx != null);

        BigInteger value = tx.value;
        if (value.compareTo(BigInteger.ZERO) < 0) {
            return new Result(Status.InvalidParameter, tx.energyLimit, "bad value");
        }

        if (tx.isCreate) {
            if (!task.getThisTransactionalKernel().isValidEnergyLimitForCreate(tx.energyLimit)) {
                return new Result(Status.InvalidParameter, tx.energyLimit, "bad step limit for create");
            }
        } else {
            if (!task.getThisTransactionalKernel().isValidEnergyLimitForNonCreate(tx.energyLimit)) {
                return new Result(Status.InvalidParameter, tx.energyLimit, "bad step limit for call");
            }
        }
        return runCommon(task.getThisTransactionalKernel(), tx);
    }

    private Result runCommon(IExternalState kernel, Transaction tx) {
        /*
         * Run the common logic with the parent kernel as the top-level one.
         * After this point, no rejection should occur.
         */
        // start with the successful result
        Result result;

        AionAddress senderAddress = tx.senderAddress;
        AionAddress recipient = tx.destinationAddress;

        if (tx.isCreate) {
            logger.trace("=== DAppCreator ===");
            result = DAppCreator.create(kernel, task,
                    senderAddress, recipient, tx, 0,
                    this.preserveDebuggability, this.enableVerboseContractErrors, this.enableBlockchainPrintln);
        } else {
            LoadedDApp dapp;

            // See if this call is trying to reenter one already on this call-stack.
            ReentrantDAppStack.ReentrantState stateToResume = task.getReentrantDAppStack().tryShareState(recipient);

            if (null != stateToResume) {
                dapp = stateToResume.dApp;
            } else {
                try {
                    dapp = loader.load(recipient, kernel, preserveDebuggability);
                } catch (IOException e) {
                    throw RuntimeAssertionError.unexpected(e);
                }
            }
            logger.trace("=== DAppExecutor ===");
            result = DAppExecutor.call(kernel, dapp, stateToResume, task,
                    senderAddress, recipient, tx, 0,
                    this.enableVerboseContractErrors, this.enableBlockchainPrintln);
        }

        if (result.getStatus()==Status.Success) {
            kernel.commit();
        }

        task.outputFlush();
        return result;
    }

    public void shutdown() {
        InstrumentationHelpers.detachThread(instrumentation);
        instrumentationFactory.destroyInstrumentation(instrumentation);
    }
}
