package org.aion.avm.core;

import foundation.icon.ee.score.Loader;
import foundation.icon.ee.types.DAppRuntimeState;
import i.RuntimeAssertionError;
import org.aion.avm.core.persistence.LoadedDApp;
import org.aion.avm.core.util.ByteArrayWrapper;

import java.util.ArrayDeque;
import java.util.Arrays;
import java.util.Deque;
import java.util.HashMap;
import java.util.Map;
import java.util.Objects;

/**
 * Contains the state of DApps currently running within the current logical thread (DApps calling DApps) to ensure that we can properly manage
 * the state when a call back into one of these is made (since reentrant calls are permitted and must inherit the state the DApp was left in).
 * NOTE:  This is only intended to be manipulated within a single callstack.  Sharing across unrelated call stacks will cause undefined behaviour.
 * Over time, the contents stored in the ReentrantState may be moved into the LoadedDApp, since their lifecycles are closely aligned.
 */
public class ReentrantDAppStack {
    private static class LoadedDAppInfo {
        private final LoadedDApp loadedDApp;
        private final String codeID;

        public LoadedDAppInfo(LoadedDApp loadedDApp, String codeID) {
            this.loadedDApp = loadedDApp;
            this.codeID = codeID;
        }

        public LoadedDApp getLoadedDApp() {
            return loadedDApp;
        }

        public String getCodeID() {
            return codeID;
        }

        @Override
        public boolean equals(Object o) {
            if (this == o) return true;
            if (o == null || getClass() != o.getClass()) return false;
            LoadedDAppInfo that = (LoadedDAppInfo) o;
            return loadedDApp.equals(that.loadedDApp) && codeID.equals(that.codeID);
        }

        @Override
        public int hashCode() {
            return Objects.hash(loadedDApp, codeID);
        }
    }

    private final Deque<ReentrantState> stack = new ArrayDeque<>();
    private final Map<ByteArrayWrapper, LoadedDAppInfo> dAppCache = new HashMap<>();

    /**
     * Pushes the given state onto the stack.  Note that state will temporarily shadow any other states on the stack with the same address.
     * Note that this has the side-effect of making the instance loader which was previously on top "inactive".
     * 
     * @param state The new state to push.
     */
    public void pushState(ReentrantState state) {
        RuntimeAssertionError.assertTrue(null != state);

        this.stack.push(state);
        assert state.dApp != null;
        assert state.contractID != null;
        assert state.codeID != null;
        dAppCache.put(new ByteArrayWrapper(state.contractID),
                new LoadedDAppInfo(state.dApp, state.codeID));
    }

    public void pushState() {
        this.stack.push(new ReentrantState());
    }

    /**
     * Searches the stack (starting with the top) for a state with the given address, returning it (but not modifying the state of the stack)
     * if it is found.
     * 
     * @param contractID The contract ID of the state we wish to find.
     * @return The first state found with the given address.
     */
    public ReentrantState tryShareState(byte[] contractID) {
        RuntimeAssertionError.assertTrue(null != contractID);
        ReentrantState foundState = null;
        for (ReentrantState state : this.stack) {
            if (Arrays.equals(contractID, state.contractID)) {
                foundState = state;
                break;
            }
        }
        return foundState;
    }

    /**
     * Pops the top state off the stack and returns it.  Returns null if the stack is empty.
     * Note that this has the side-effect of making the instance loader which is newly on top "active".
     * 
     * @return The state which was previously on top of the stack (null if empty).
     */
    public ReentrantState popState() {
        return (this.stack.isEmpty())
                ? null
                : this.stack.pop();
    }

    public DAppRuntimeState getRuntimeState(int eid) {
        // top to bottom
        for (ReentrantState reentrantState : stack) {
            var rs = reentrantState.getRuntimeState(eid);
            if (rs != null) {
                return rs;
            }
        }
        return null;
    }

    public ReentrantState getTop() {
        return this.stack.peekFirst();
    }


    public static class ReentrantState {
        public final LoadedDApp dApp;
        public final byte[] contractID;
        public final String codeID;
        private final Map<Integer, SaveItem> saveItems = new HashMap<>();

        public ReentrantState() {
            this.dApp = null;
            this.contractID = null;
            this.codeID = null;
        }

        public ReentrantState(LoadedDApp dApp, byte[] contractID, String codeID) {
            this.dApp = dApp;
            this.contractID = contractID;
            this.codeID = codeID;
        }

        public DAppRuntimeState getRuntimeState(int eid) {
            var saveItem = saveItems.get(eid);
            if (saveItem == null) {
                return null;
            }
            return saveItem.getRuntimeState();
        }

        public void setRuntimeState(int eid, DAppRuntimeState rs, byte[] contractID) {
            saveItems.put(eid, new SaveItem(rs, contractID));
        }

        public void removeRuntimeStatesByAddress(byte[] contractID) {
            this.saveItems.entrySet().removeIf((si) ->
                    Arrays.equals(si.getValue().getContractID(), contractID)
            );
        }

        void inherit(ReentrantState s) {
            saveItems.putAll(s.saveItems);
        }
    }

    static class SaveItem {
        private final DAppRuntimeState runtimeState;
        private final byte[] contractID;

        public SaveItem(DAppRuntimeState runtimeState, byte[] contractID) {
            this.runtimeState = runtimeState;
            this.contractID = contractID;
        }

        public DAppRuntimeState getRuntimeState() {
            return runtimeState;
        }

        public byte[] getContractID() {
            return contractID;
        }
    }

    public LoadedDApp tryGetLoadedDApp(byte[] contractID) {
        var dAppInfo = dAppCache.get(new ByteArrayWrapper(contractID));
        return  dAppInfo != null ? dAppInfo.getLoadedDApp() : null;
    }

    public void cacheDApp(LoadedDApp dApp, byte[] contractID, String codeID) {
        dAppCache.put(new ByteArrayWrapper(contractID),
                new LoadedDAppInfo(dApp, codeID));
    }

    public void unloadDApps(Loader loader) {
        for (var e : dAppCache.entrySet()) {
            loader.unload(e.getValue().getCodeID(),
                    e.getValue().getLoadedDApp());
        }
    }
}
