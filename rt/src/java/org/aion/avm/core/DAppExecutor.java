package org.aion.avm.core;

import org.aion.avm.core.util.TransactionResultUtil;
import org.aion.kernel.AvmWrappedTransactionResult;
import org.aion.kernel.AvmWrappedTransactionResult.AvmInternalError;
import org.aion.types.AionAddress;
import org.aion.types.Transaction;
import org.aion.avm.RuntimeMethodFeeSchedule;
import org.aion.avm.StorageFees;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.avm.core.persistence.ReentrantGraph;
import org.aion.avm.core.util.Helpers;
import i.*;
import org.aion.parallel.TransactionTask;


public class DAppExecutor {
    public static AvmWrappedTransactionResult call(IExternalCapabilities capabilities, IExternalState externalState, AvmInternal avm, LoadedDApp dapp,
                            ReentrantDAppStack.ReentrantState stateToResume, TransactionTask task,
                            Transaction tx, AvmWrappedTransactionResult internalResult, boolean verboseErrors, boolean readFromCache, boolean enableBlockchainPrintln) {
        AvmWrappedTransactionResult result = internalResult;
        AionAddress dappAddress = tx.destinationAddress;
        
        // If this is a reentrant call, we need to serialize the graph of the parent frame.  This is required to both copy-back our changes but also
        // is required in case we want to revert the state.
        ReentrantGraph callerState = (null != stateToResume)
                ? dapp.captureStateAsCaller(stateToResume.getNextHashCode(), StorageFees.MAX_GRAPH_SIZE)
                : null;
        
        // Note that the instrumentation is just a per-thread access to the state stack - we can grab it at any time as it never changes for this thread.
        IInstrumentation threadInstrumentation = IInstrumentation.attachedThreadInstrumentation.get();
        
        // We need to get the interned classes before load the graph since it might need to instantiate class references.
        InternedClasses initialClassWrappers = (null != stateToResume)
            ? stateToResume.getInternedClassWrappers()
            : new InternedClasses();

        // Note that we can't do any billing until after we install the InstrumentationHelpers new stack frame
        int nextHashCode;
        // Used for deserialization billing
        int rawGraphDataLength;

        if (readFromCache) {
            if (null != callerState) {
                nextHashCode = stateToResume.getNextHashCode();
                byte[] rawGraphData = callerState.rawState;
                dapp.loadEntireGraph(initialClassWrappers, rawGraphData);
                rawGraphDataLength = rawGraphData.length;

            } else {
                // If we have the DApp in cache, we can get the next hashcode and graph length from it. Otherwise, we have to load the entire graph
                nextHashCode = dapp.getHashCode();
                rawGraphDataLength = dapp.getSerializedLength();
            }
        } else {
            byte[] rawGraphData = (null != callerState)
                    ? callerState.rawState
                    : externalState.getObjectGraph(dappAddress);
            nextHashCode = dapp.loadEntireGraph(initialClassWrappers, rawGraphData);
            rawGraphDataLength = rawGraphData.length;
        }


        // Note that we need to store the state of this invocation on the reentrant stack in case there is another call into the same app.
        // This is required so that the call() mechanism can access it to save/reload its ContractEnvironmentState and so that the underlying
        // instance loader (ReentrantGraphProcessor/ReflectionStructureCodec) can be notified when it becomes active/inactive (since it needs
        // to know if it is loading an instance
        ReentrantDAppStack.ReentrantState thisState = new ReentrantDAppStack.ReentrantState(dappAddress, dapp, nextHashCode, initialClassWrappers);
        task.getReentrantDAppStack().pushState(thisState);

        InstrumentationHelpers.pushNewStackFrame(dapp.runtimeSetup, dapp.loader, tx.energyLimit - result.energyUsed(), nextHashCode, initialClassWrappers);
        IBlockchainRuntime previousRuntime = dapp.attachBlockchainRuntime(new BlockchainRuntimeImpl(capabilities, externalState, avm, thisState, task, tx, tx.copyOfTransactionData(), dapp.runtimeSetup, enableBlockchainPrintln));

        try {
            // It is now safe for us to bill for the cost of loading the graph (the cost is the same, whether this came from the caller or the disk).
            // (note that we do this under the try since aborts can happen here)
            threadInstrumentation.chargeEnergy(StorageFees.READ_PRICE_PER_BYTE * rawGraphDataLength);
            
            // Call the main within the DApp.
            byte[] ret = dapp.callMain();

            // Save back the state before we return.
            if (null != stateToResume) {
                int updatedNextHashCode = threadInstrumentation.peekNextHashCode();
                ReentrantGraph calleeState = dapp.captureStateAsCallee(updatedNextHashCode, StorageFees.MAX_GRAPH_SIZE);
                // Bill for writing this size.
                threadInstrumentation.chargeEnergy(StorageFees.WRITE_PRICE_PER_BYTE * calleeState.rawState.length);
                // Now, commit this back into the callerState.
                dapp.commitReentrantChanges(initialClassWrappers, callerState, calleeState);
                // Update the final hash code.
                stateToResume.updateNextHashCode(updatedNextHashCode);
            } else {
                // We are at the "top" so write this back to disk.
                int newHashCode = threadInstrumentation.peekNextHashCode();
                byte[] postCallGraphData = dapp.saveEntireGraph(newHashCode, StorageFees.MAX_GRAPH_SIZE);
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
                // refund is calculated for the transaction if it included a selfdestruct operation or it set the storage value from nonzero to zero
                long selfDestructRefund = 0l;
                long resetStorageRefund = 0l;

                if (task.getSelfDestructAddressCount() > 0) {
                    selfDestructRefund = task.getSelfDestructAddressCount() * RuntimeMethodFeeSchedule.BlockchainRuntime_avm_selfDestruct_refund;
                }
                if (task.getResetStorageKeyCount() > 0) {
                    resetStorageRefund = task.getResetStorageKeyCount() * RuntimeMethodFeeSchedule.BlockchainRuntime_avm_deleteStorage_refund;
                }
                // refund is capped at half the energy used for the whole transaction
                refund = Math.min(energyUsed / 2, selfDestructRefund + resetStorageRefund);
            }

            result = TransactionResultUtil.setEnergyUsed(result, energyUsed - refund);
        } catch (OutOfEnergyException e) {
            if (verboseErrors) {
                System.err.println("DApp execution failed due to Out-of-Energy EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            if (null != stateToResume) {
                dapp.revertToCallerState(initialClassWrappers, callerState);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_OUT_OF_ENERGY, tx.energyLimit);

        } catch (OutOfStackException e) {
            if (verboseErrors) {
                System.err.println("DApp execution failed due to stack overflow EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            if (null != stateToResume) {
                dapp.revertToCallerState(initialClassWrappers, callerState);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_OUT_OF_STACK, tx.energyLimit);

        } catch (CallDepthLimitExceededException e) {
            if (verboseErrors) {
                System.err.println("DApp execution failed due to call depth limit EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            if (null != stateToResume) {
                dapp.revertToCallerState(initialClassWrappers, callerState);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_CALL_DEPTH_LIMIT, tx.energyLimit);

        } catch (RevertException e) {
            if (verboseErrors) {
                System.err.println("DApp execution to REVERT due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            if (null != stateToResume) {
                dapp.revertToCallerState(initialClassWrappers, callerState);
            }
            result = TransactionResultUtil.setRevertedFailureAndEnergyUsed(result, tx.energyLimit - threadInstrumentation.energyLeft());

        } catch (InvalidException e) {
            if (verboseErrors) {
                System.err.println("DApp execution INVALID due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            if (null != stateToResume) {
                dapp.revertToCallerState(initialClassWrappers, callerState);
            }
            result = TransactionResultUtil.setNonRevertedFailureAndEnergyUsed(result, AvmInternalError.FAILED_INVALID, tx.energyLimit);

        } catch (EarlyAbortException e) {
            if (verboseErrors) {
                System.err.println("FYI - concurrent abort (will retry) in transaction \"" + Helpers.bytesToHexString(tx.copyOfTransactionHash()) + "\"");
            }
            if (null != stateToResume) {
                dapp.revertToCallerState(initialClassWrappers, callerState);
            }
            result = TransactionResultUtil.newAbortedResultWithZeroEnergyUsed();

        } catch (UncaughtException e) {
            if (verboseErrors) {
                System.err.println("DApp execution failed due to uncaught EXCEPTION: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            if (null != stateToResume) {
                dapp.revertToCallerState(initialClassWrappers, callerState);
            }
            result = TransactionResultUtil.setFailedException(result, e.getCause(), tx.energyLimit);
        } catch (AvmException e) {
            // We handle the generic AvmException as some failure within the contract.
            if (verboseErrors) {
                System.err.println("DApp execution failed due to AvmException: \"" + e.getMessage() + "\"");
                e.printStackTrace(System.err);
            }
            if (null != stateToResume) {
                dapp.revertToCallerState(initialClassWrappers, callerState);
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
