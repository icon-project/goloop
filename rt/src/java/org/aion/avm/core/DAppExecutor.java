package org.aion.avm.core;

import foundation.icon.ee.types.DAppRuntimeState;
import foundation.icon.ee.types.ObjectGraph;
import i.AvmException;
import i.CallDepthLimitExceededException;
import i.EarlyAbortException;
import i.IBlockchainRuntime;
import i.IInstrumentation;
import i.InstrumentationHelpers;
import i.InternedClasses;
import i.InvalidException;
import i.JvmError;
import i.OutOfEnergyException;
import i.OutOfStackException;
import i.RevertException;
import i.UncaughtException;
import org.aion.avm.RuntimeMethodFeeSchedule;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.avm.core.util.Helpers;
import org.aion.avm.core.util.TransactionResultUtil;
import org.aion.kernel.AvmWrappedTransactionResult;
import org.aion.kernel.AvmWrappedTransactionResult.AvmInternalError;
import org.aion.parallel.TransactionTask;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;

public class DAppExecutor {
    public static AvmWrappedTransactionResult call(IExternalState externalState,
                                                   LoadedDApp dapp,
                                                   ReentrantDAppStack.ReentrantState stateToResume,
                                                   TransactionTask task,
                                                   AionAddress senderAddress,
                                                   AionAddress dappAddress,
                                                   Transaction tx,
                                                   AvmWrappedTransactionResult internalResult,
                                                   boolean verboseErrors,
                                                   boolean readFromCache,
                                                   boolean enableBlockchainPrintln) {
        AvmWrappedTransactionResult result = internalResult;

        // Note that the instrumentation is just a per-thread access to the state stack - we can grab it at any time as it never changes for this thread.
        IInstrumentation threadInstrumentation = IInstrumentation.attachedThreadInstrumentation.get();
        
        // We need to get the interned classes before load the graph since it might need to instantiate class references.
        InternedClasses initialClassWrappers = dapp.getInternedClasses();

        var saveItem = task.getReentrantDAppStack().getSaveItem(dappAddress);
        DAppRuntimeState rs = null;
        if (saveItem==null) {
            var raw = externalState.getObjectGraph(dappAddress);
            var graph = ObjectGraph.getInstance(raw);
            rs = new DAppRuntimeState(null, dapp.getInternedClasses(), graph);
            saveItem = new ReentrantDAppStack.SaveItem(dapp, rs);
        } else {
            rs = saveItem.getRuntimeState();
        }
        var nextHashCode = dapp.loadRuntimeState(rs);

        // Used for deserialization billing
        int rawGraphDataLength = rs.getGraph().getGraphData().length + 4;

        // Note that we need to store the state of this invocation on the reentrant stack in case there is another call into the same app.
        // This is required so that the call() mechanism can access it to save/reload its ContractEnvironmentState and so that the underlying
        // instance loader (ReentrantGraphProcessor/ReflectionStructureCodec) can be notified when it becomes active/inactive (since it needs
        // to know if it is loading an instance
        ReentrantDAppStack.ReentrantState thisState = new ReentrantDAppStack.ReentrantState(dappAddress, dapp, nextHashCode);
        var prevState = task.getReentrantDAppStack().getTop();
        task.getReentrantDAppStack().pushState(thisState);

        IBlockchainRuntime br = new BlockchainRuntimeImpl(externalState,
                                                          thisState,
                                                          task,
                                                          senderAddress,
                                                          dappAddress,
                                                          tx,
                                                          dapp.runtimeSetup,
                                                          dapp,
                                                          enableBlockchainPrintln);
        FrameContextImpl fc = new FrameContextImpl(externalState, dapp, initialClassWrappers, br);
        InstrumentationHelpers.pushNewStackFrame(dapp.runtimeSetup, dapp.loader, tx.energyLimit - result.energyUsed(), nextHashCode, initialClassWrappers, fc);
        IBlockchainRuntime previousRuntime = dapp.attachBlockchainRuntime(br);

        try {
            // It is now safe for us to bill for the cost of loading the graph (the cost is the same, whether this came from the caller or the disk).
            // (note that we do this under the try since aborts can happen here)
            threadInstrumentation.chargeEnergy(StorageFees.READ_PRICE_PER_BYTE * rawGraphDataLength);
            
            // Call the main within the DApp.
            Object ret = dapp.callMethod(tx.method, tx.getParams());

            var runtimeState = dapp.saveRuntimeState();

            // Save back the state before we return.
            if (null == stateToResume) {
                // We are at the "top" so write this back to disk.
                int newHashCode = threadInstrumentation.peekNextHashCode();
                byte[] postCallGraphData = runtimeState.getGraph().getRawData();
                // Bill for writing this size.
                threadInstrumentation.chargeEnergy(StorageFees.WRITE_PRICE_PER_BYTE * postCallGraphData.length);
                externalState.putObjectGraph(dappAddress, postCallGraphData);
                // Update LoadedDApp state at the end of execution
                dapp.setHashCode(newHashCode);
                dapp.setSerializedLength(postCallGraphData.length);
            }

            result = TransactionResultUtil.setSuccessfulOutput(result, ret);
            long refund = 0;
            long energyUsed = tx.energyLimit - threadInstrumentation.energyLeft();
            //refund is only calculated for the external transaction
            if (task.getTransactionStackDepth() == 0) {
                // refund is calculated for the transaction if it set the storage value from nonzero to zero
                long resetStorageRefund = 0L;

                if (task.getResetStorageKeyCount() > 0) {
                    resetStorageRefund = task.getResetStorageKeyCount() * RuntimeMethodFeeSchedule.BlockchainRuntime_avm_deleteStorage_refund;
                }
                // refund is capped at half the energy used for the whole transaction
                refund = Math.min(energyUsed / 2, resetStorageRefund);
            }
            result = TransactionResultUtil.setEnergyUsed(result, energyUsed - refund);
            if (prevState != null) {
                prevState.getSaveItems().putAll(thisState.getSaveItems());
                prevState.getSaveItems().put(dappAddress, new ReentrantDAppStack.SaveItem(dapp, runtimeState));
            }
        } catch (OutOfEnergyException e) {
            if (verboseErrors) {
                System.err.println("DApp execution failed due to Out-of-Energy EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_OUT_OF_ENERGY, tx.energyLimit);

        } catch (OutOfStackException e) {
            if (verboseErrors) {
                System.err.println("DApp execution failed due to stack overflow EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_OUT_OF_STACK, tx.energyLimit);

        } catch (CallDepthLimitExceededException e) {
            if (verboseErrors) {
                System.err.println("DApp execution failed due to call depth limit EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_CALL_DEPTH_LIMIT, tx.energyLimit);

        } catch (RevertException e) {
            if (verboseErrors) {
                System.err.println("DApp execution to REVERT due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setRevertedFailureAndEnergyUsed(result, tx.energyLimit - threadInstrumentation.energyLeft());

        } catch (InvalidException e) {
            if (verboseErrors) {
                System.err.println("DApp execution INVALID due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_INVALID, tx.energyLimit);

        } catch (EarlyAbortException e) {
            if (verboseErrors) {
                System.err.println("FYI - concurrent abort (will retry) in transaction \"" + Helpers.bytesToHexString(tx.copyOfTransactionHash()) + "\"");
            }
            result = TransactionResultUtil.newAbortedResultWithZeroEnergyUsed();

        } catch (UncaughtException e) {
            if (verboseErrors) {
                System.err.println("DApp execution failed due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setFailedException(result, e.getCause(), tx.energyLimit);
        } catch (AvmException e) {
            // We handle the generic AvmException as some failure within the contract.
            if (verboseErrors) {
                System.err.println("DApp execution failed due to AvmException: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED, tx.energyLimit);
        } catch (JvmError e) {
            // These are cases which we know we can't handle and have decided to handle by safely stopping the AVM instance so
            // re-throw this as the AvmImpl top-level loop will commute it into an asynchronous shutdown.
            if (verboseErrors) {
                System.err.println("FATAL JvmError: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            throw e;
        } catch (Throwable e) {
            // We don't know what went wrong in this case, but it is beyond our ability to handle it here.
            // We ship it off to the ExceptionHandler, which kills the transaction as a failure for unknown reasons.
            System.err.println("Exception on method " + tx.method);
            result = DAppExceptionHandler.handle(e, result, tx.energyLimit, verboseErrors);
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
