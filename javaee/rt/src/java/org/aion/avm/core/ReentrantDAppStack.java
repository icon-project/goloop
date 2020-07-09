package org.aion.avm.core;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.DAppRuntimeState;
import i.RuntimeAssertionError;
import org.aion.avm.core.persistence.LoadedDApp;

import java.util.ArrayDeque;
import java.util.Deque;
import java.util.HashMap;
import java.util.Map;

/**
 * Contains the state of DApps currently running within the current logical thread (DApps calling DApps) to ensure that we can properly manage
 * the state when a call back into one of these is made (since reentrant calls are permitted and must inherit the state the DApp was left in).
 * NOTE:  This is only intended to be manipulated within a single callstack.  Sharing across unrelated call stacks will cause undefined behaviour.
 * Over time, the contents stored in the ReentrantState may be moved into the LoadedDApp, since their lifecycles are closely aligned.
 */
public class ReentrantDAppStack {
    private final Deque<ReentrantState> stack = new ArrayDeque<>();

    /**
     * Pushes the given state onto the stack.  Note that state will temporarily shadow any other states on the stack with the same address.
     * Note that this has the side-effect of making the instance loader which was previously on top "inactive".
     * 
     * @param state The new state to push.
     */
    public void pushState(ReentrantState state) {
        RuntimeAssertionError.assertTrue(null != state);

        this.stack.push(state);
    }

    public void pushState() {
        this.stack.push(new ReentrantState());
    }

    /**
     * Searches the stack (starting with the top) for a state with the given address, returning it (but not modifying the state of the stack)
     * if it is found.
     * 
     * @param address The address of the state we wish to find.
     * @return The first state found with the given address.
     */
    public ReentrantState tryShareState(Address address) {
        RuntimeAssertionError.assertTrue(null != address);
        ReentrantState foundState = null;
        for (ReentrantState state : this.stack) {
            if (address.equals(state.address)) {
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
        public final Address address;
        public final LoadedDApp dApp;
        private final Map<Integer, SaveItem> saveItems = new HashMap<>();

        public ReentrantState() {
            this.address = null;
            this.dApp = null;
        }

        public ReentrantState(Address address, LoadedDApp dApp) {
            this.address = address;
            this.dApp = dApp;
        }

        public DAppRuntimeState getRuntimeState(int eid) {
            var saveItem = saveItems.get(eid);
            if (saveItem == null) {
                return null;
            }
            return saveItem.getRuntimeState();
        }

        public void setRuntimeState(int eid, DAppRuntimeState rs, Address addr) {
            saveItems.put(eid, new SaveItem(rs, addr));
        }

        public void removeRuntimeStatesByAddress(Address address) {
            this.saveItems.entrySet().removeIf((si) ->
                    si.getValue().getAddress().equals(address)
            );
        }

        void inherit(ReentrantState s) {
            saveItems.putAll(s.saveItems);
        }
    }

    static class SaveItem {
        private final DAppRuntimeState runtimeState;
        private final Address address;

        public SaveItem(DAppRuntimeState runtimeState, Address address) {
            this.runtimeState = runtimeState;
            this.address = address;
        }

        public DAppRuntimeState getRuntimeState() {
            return runtimeState;
        }

        public Address getAddress() {
            return address;
        }
    }
}
