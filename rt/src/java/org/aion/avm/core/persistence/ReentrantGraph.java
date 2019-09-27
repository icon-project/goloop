package org.aion.avm.core.persistence;

import java.nio.ByteBuffer;
import java.util.ArrayList;
import java.util.List;


/**
 * Contains both the data and logic required to capture, revert, and commit a reentrant invocation.
 * In the future, this logic and data may be split, since they don't need to be together.  This just makes the connection more obvious, for now.
 */
public class ReentrantGraph {
    public static ReentrantGraph captureCallerState(IGlobalResolver resolver, SortedFieldCache cache, IPersistenceNameMapper classNameMapper, int maximumSizeInBytes, int nextHashCode, Class<?>[] sortedRoots, Class<?> constantClass) {
        ByteBuffer buffer = ByteBuffer.allocate(maximumSizeInBytes);
        List<Object> existingObjectIndex = new ArrayList<>();
        Serializer.serializeEntireGraph(buffer, existingObjectIndex, null, resolver, cache, classNameMapper, nextHashCode, sortedRoots, constantClass);
        byte[] finalBytes = new byte[buffer.position()];
        System.arraycopy(buffer.array(), 0, finalBytes, 0, finalBytes.length);
        return new ReentrantGraph(finalBytes, existingObjectIndex, null);
    }

    public static ReentrantGraph captureCalleeState(IGlobalResolver resolver, SortedFieldCache cache, IPersistenceNameMapper classNameMapper, int maximumSizeInBytes, int nextHashCode, Class<?>[] sortedRoots, Class<?> constantClass) {
        ByteBuffer calleeBuffer = ByteBuffer.allocate(maximumSizeInBytes);
        List<Integer> calleeToCallerMapping = new ArrayList<>();
        Serializer.serializeEntireGraph(calleeBuffer, null, calleeToCallerMapping, resolver, cache, classNameMapper, nextHashCode, sortedRoots, constantClass);
        byte[] calleeBytes = new byte[calleeBuffer.position()];
        System.arraycopy(calleeBuffer.array(), 0, calleeBytes, 0, calleeBytes.length);
        return new ReentrantGraph(calleeBytes, null, calleeToCallerMapping);
    }


    public final byte[] rawState;
    private final List<Object> existingObjectIndex;
    private final List<Integer> calleeToCallerMapping;

    public ReentrantGraph(byte[] rawState, List<Object> existingObjectIndex, List<Integer> calleeToCallerMapping) {
        this.rawState = rawState;
        this.existingObjectIndex = existingObjectIndex;
        this.calleeToCallerMapping = calleeToCallerMapping;
    }

    public int commitChangesToState(IGlobalResolver resolver, SortedFieldCache cache, IPersistenceNameMapper classNameMapper, Class<?>[] sortedRoots, Class<?> constantClass, ReentrantGraph calleeState) {
        // Now for the interesting logic (which needs to be moved out of this test-case since it is real functionality): remapping the caller index.
        List<Object> updatedIndex = new ArrayList<>();
        for (int callerIndex : calleeState.calleeToCallerMapping) {
            Object target = (-1 == callerIndex)
                    ? null
                    : existingObjectIndex.get(callerIndex);
            updatedIndex.add(target);
        }
        
        ByteBuffer readingBuffer = ByteBuffer.wrap(calleeState.rawState);
        return Deserializer.deserializeEntireGraphAndNextHashCode(readingBuffer, updatedIndex, resolver, cache, classNameMapper, sortedRoots, constantClass);
    }

    public int revertChangesToState(IGlobalResolver resolver, SortedFieldCache cache, IPersistenceNameMapper classNameMapper, Class<?>[] sortedRoots, Class<?> constantClass) {
        // We just re-apply what we already captured from last time.
        ByteBuffer readingBuffer = ByteBuffer.wrap(this.rawState);
        return Deserializer.deserializeEntireGraphAndNextHashCode(readingBuffer, this.existingObjectIndex, resolver, cache, classNameMapper, sortedRoots, constantClass);
    }

    public int applyToRootsForNewFrame(IGlobalResolver resolver, SortedFieldCache cache, IPersistenceNameMapper classNameMapper, Class<?>[] sortedRoots, Class<?> constantClass) {
        ByteBuffer fakeBuffer = ByteBuffer.wrap(this.rawState);
        return Deserializer.deserializeEntireGraphAndNextHashCode(fakeBuffer, null, resolver, cache, classNameMapper, sortedRoots, constantClass);
    }
}
