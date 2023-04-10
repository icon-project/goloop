package i;

public class StackWatcher {

    /* StackWatcher policy:
     *  POLICY_DEPTH will keep JVM stack within depth of maxStackDepth.
     *  POLICY_SIZE  will keep JVM stack within size (in terms of JVM stack
     *  frame slots) of maxStackSize. With Java 10 each slot is 8 bytes.
     *  (POLICY_DEPTH | POLICY_SIZE) will enforce both policy
     */
    public static final int POLICY_DEPTH = 1;
    public static final int POLICY_SIZE  = 1 << 1;

    // Reserved stack frame slot for AVM internal use
    private static final int RESERVED_AVM_SLOT = 10;
    // Reserved stack frame slot for JVM internal use
    private static final int RESERVED_JVM_SLOT = 10;

    private boolean checkDepth = false;
    private boolean checkSize  = false;

    private int maxStackDepth = 200;
    private int maxStackSize  = 100000;

    private int curDepth = 0;
    private int curSize  = 0;

    /**
     * Set the policy of current stack watcher
     * @param policy A policy mask. See AVMStackWatcher.POLICY_DEPTH and AVMStackWatcher.POLICY_Size.
     */
    public void setPolicy(int policy){
        checkDepth = (policy & POLICY_DEPTH) == POLICY_DEPTH;
        checkSize  = (policy & POLICY_SIZE)  == POLICY_SIZE;
    }

    public void reset(){
        curDepth = 0;
        curSize = 0;
    }

    /**
     * Get the current stack size (as number of slots).
     * @return current stack size.
     */
    public int getCurStackSize(){
        return curSize;
    }

    /**
     * Get the current stack depth.
     * @return current stack depth.
     */
    public int getCurStackDepth(){
        return curDepth;
    }

    /**
     * Set the stack size limit (as number of slots).
     * @param limit new stack size limit.
     */
    public void setMaxStackDepth(int limit){
        maxStackDepth = limit;
    }

    /**
     * Get the stack depth limit.
     * @return current stack depth limit.
     */
    public int getMaxStackDepth(){
        return maxStackDepth;
    }

    /**
     * Set the stack size limit (as number of slots).
     * @param limit new stack size limit.
     */
    public void setMaxStackSize(int limit){
        maxStackSize = limit;
    }

    /**
     * Get the stack size limit (as number of slots).
     * @return current stack size limit.
     */
    public int getMaxStackSize(){
        return maxStackSize;
    }

    /**
     * This method will be inserted into the beginning of every instrumented method.
     * It will validate/advance the depth and size of the current JVM stack.
     * Abort the smart contract in case of overflow.
     * @param frameSize size of the current frame (in number of slots).
     */
    public void enterMethod(int frameSize) throws OutOfStackException {
        if (checkDepth && (curDepth++ > maxStackDepth)){
            abortCurrentContract();
        }

        frameSize += RESERVED_AVM_SLOT + RESERVED_JVM_SLOT;
        if (checkSize && ( (curSize = curSize + frameSize) > maxStackSize)){
            abortCurrentContract();
        }
    }

    /**
     * This method will be inserted into every exit point of every instrumented method.
     * It will validate/shrink the depth and size of the current JVM stack.
     * Abort the smart contract in case of underflow.
     * @param frameSize size of the current frame (in number of slots).
     */
    public void exitMethod(int frameSize) throws OutOfStackException {
        if (checkDepth && (curDepth-- < 0)){
            abortCurrentContract();
        }

        frameSize += RESERVED_AVM_SLOT + RESERVED_JVM_SLOT;
        if (checkSize && ((curSize = curSize - frameSize) < 0)){
            abortCurrentContract();
        }
    }

    /**
     * This method will be inserted into the beginning of every catch block.
     * If a method contains try catch block(s), we generate a stack watcher stamp.
     * The stamp will be stored as local variables of the instrumented method.
     * In case of an exception caught, we load the stamp to get the correct depth and size.
     * @param depth stack depth from the stamp.
     * @param size stack size from the stamp.
     */
    public void enterCatchBlock(int depth, int size){
        curDepth = depth;
        curSize = size;
    }

    private void abortCurrentContract() throws OutOfStackException {
        throw new OutOfStackException();
    }
}
