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
import foundation.icon.ee.types.Transaction;
import i.AvmException;
import i.IInstrumentation;
import i.IInstrumentationFactory;
import i.InstrumentationHelpers;
import org.aion.avm.core.AvmConfiguration;
import org.aion.avm.core.DAppCreator;
import org.aion.avm.core.DAppExecutor;
import org.aion.avm.core.IExternalState;
import org.aion.avm.core.ReentrantDAppStack;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.parallel.TransactionTask;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class AvmExecutor {
    private static final Logger logger = LoggerFactory.getLogger(AvmExecutor.class);

    private final IInstrumentationFactory instrumentationFactory;
    private final AvmConfiguration conf;
    private final Loader loader;
    private IInstrumentation instrumentation;
    private TransactionTask task;

    public AvmExecutor(IInstrumentationFactory factory, AvmConfiguration conf, Loader loader) {
        this.instrumentationFactory = factory;
        this.conf = new AvmConfiguration(conf);
        this.loader = loader;
    }

    public void start() {
        instrumentation = instrumentationFactory.createInstrumentation();
        InstrumentationHelpers.attachThread(instrumentation);
    }

    public Result run(IExternalState kernel, Transaction transaction,
            Address origin, int eid, int prevEID) {
        if (task == null) {
            return runExternal(kernel, transaction, origin, eid, prevEID);
        } else {
            return runInternal(kernel, transaction, eid, prevEID);
        }
    }

    private Result runExternal(IExternalState kernel, Transaction transaction,
            Address origin, int eid, int prevEID) {
        // Get the first task
        task = new TransactionTask(kernel, transaction, origin);

        task.startNewTransaction();
        task.attachInstrumentationForThread();
        Result result;
        try {
            result = runCommon(task.getThisTransactionalKernel(),
                    transaction, eid, prevEID);
        } finally {
            task.getReentrantDAppStack().unloadDApps(loader);
            task.detachInstrumentationForThread();
        }

        logger.trace("{}", result);
        task = null;
        return result;
    }

    private Result runInternal(IExternalState kernel, Transaction transaction,
            int eid, int prevEID) {
        return runCommon(kernel, transaction, eid, prevEID);
    }

    private Result runCommon(IExternalState kernel, Transaction tx, int eid,
            int prevEID) {
        Address senderAddress = tx.getSender();
        Address recipient = tx.getDestination();

        task.setEID(eid);
        task.setPrevEID(prevEID);

        if (tx.isCreate()) {
            logger.trace("=== DAppCreator ===");
            return DAppCreator.create(kernel, task, senderAddress, recipient,
                    tx, conf);
        } else {
            // See if this call is trying to reenter one already on this call-stack.
            var cid = kernel.getContractID();
            ReentrantDAppStack.ReentrantState stateToResume = task.getReentrantDAppStack().tryShareState(cid);
            LoadedDApp dapp = task.getReentrantDAppStack().tryGetLoadedDApp(cid);
            if (dapp == null) {
                try {
                    dapp = loader.load(kernel, conf);
                } catch (AvmException e) {
                    return new Result(e.getCode(), 0, e.getResultMessage());
                }
            }
            return DAppExecutor.call(kernel, dapp, stateToResume, task,
                    senderAddress, recipient, tx, conf);
        }
    }

    public void shutdown() {
        InstrumentationHelpers.detachThread(instrumentation);
        instrumentationFactory.destroyInstrumentation(instrumentation);
    }
}
