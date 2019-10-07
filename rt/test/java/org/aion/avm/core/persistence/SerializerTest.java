package org.aion.avm.core.persistence;

import org.aion.avm.core.util.Helpers;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import i.RuntimeAssertionError;
import org.junit.Assert;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Test;

import java.lang.reflect.Field;
import java.lang.reflect.Method;
import java.nio.ByteBuffer;


public class SerializerTest {
    // NOTE:  Output is ONLY produced if REPORT is set to true.
    private static final boolean REPORT = false;

    private static TargetLeaf TEST_CONSTANT;
    private SortedFieldCache cache;

    @BeforeClass
    public static void setupClass() throws Exception {
        TEST_CONSTANT = new TargetLeaf();
        TEST_CONSTANT.counter = -1;
    }

    @Before
    public void setup() throws Exception {
        Method serializeSelf = TargetRoot.class.getMethod("serializeSelf", Class.class, IObjectSerializer.class);
        Method deserializeSelf = TargetRoot.class.getMethod("deserializeSelf", Class.class, IObjectDeserializer.class);
        Field readIndex = TargetRoot.class.getField("readIndex");
        this.cache = new SortedFieldCache(SerializerTest.class.getClassLoader(), serializeSelf, deserializeSelf, readIndex);
    }

    @Test
    public void testSimpleCase() throws Exception {
        ByteBuffer buffer = ByteBuffer.allocate(1000);
        TestGlobalResolver resolver = new TestGlobalResolver();
        TestNameMapper classNameMapper = new TestNameMapper();
        
        TargetRoot.root = new TargetRoot();
        TargetRoot.root.counter = 1;
        TargetLeaf next = new TargetLeaf();
        next.counter = 2;
        next.left = TargetRoot.root;
        next.right = next;
        TargetRoot.root.next = next;
        
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetRoot.class, TargetLeaf.class};
        byte[] finalBytes = serializeDeserializeAsNew(nextHashCode, sortedRoots);
        Serializer.serializeEntireGraph(buffer, null, null, resolver, this.cache, classNameMapper, nextHashCode, sortedRoots, EmptyConstantClass.class);
        Assert.assertArrayEquals(Helpers.hexStringToBytes("00000001030000000000000000000000000a546172676574526f6f740000000103000000010a5461726765744c656166000000020003000000000300000001"), finalBytes);
        
        Assert.assertEquals(1, TargetRoot.root.counter);
        TargetLeaf checkNext = (TargetLeaf) TargetRoot.root.next;
        Assert.assertEquals(2, checkNext.counter);
        Assert.assertTrue(TargetRoot.root == checkNext.left);
        Assert.assertTrue(checkNext == checkNext.right);
    }

    @Test
    public void testWithNulls() throws Exception {
        TargetRoot.root = new TargetRoot();
        TargetRoot.root.counter = 1;
        TargetLeaf next = new TargetLeaf();
        next.counter = 2;
        next.left = null;
        next.right = next;
        TargetRoot.root.next = next;
        
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetRoot.class, TargetLeaf.class};
        byte[] finalBytes = serializeDeserializeAsNew(nextHashCode, sortedRoots);
        Assert.assertArrayEquals(Helpers.hexStringToBytes("00000001030000000000000000000000000a546172676574526f6f740000000103000000010a5461726765744c6561660000000200000300000001"), finalBytes);
        
        Assert.assertEquals(1, TargetRoot.root.counter);
        TargetLeaf checkNext = (TargetLeaf) TargetRoot.root.next;
        Assert.assertEquals(2, checkNext.counter);
        Assert.assertTrue(null == checkNext.left);
        Assert.assertTrue(checkNext == checkNext.right);
    }

    @Test
    public void testWithArray() throws Exception {
        TargetRoot left = new TargetRoot();
        left.counter = 2;
        TargetRoot right = new TargetRoot();
        right.counter = 3;
        TargetArray array = new TargetArray(6);
        TargetRoot.root = array;
        TargetRoot.root.counter = 1;
        for (int i = 0; i < 6; ++i) {
            TargetLeaf leaf = new TargetLeaf();
            leaf.counter = 4 + i;
            leaf.left = left;
            leaf.right = right;
            array.array[i] = leaf;
        }
        
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetRoot.class, TargetLeaf.class, TargetArray.class};
        byte[] finalBytes = serializeDeserializeAsNew(nextHashCode, sortedRoots);
        Assert.assertArrayEquals(Helpers.hexStringToBytes("00000001030000000000000000000000000b54617267657441727261790000000100000000060300000001030000000203000000030300000004030000000503000000060a5461726765744c6561660000000400030000000703000000080a5461726765744c6561660000000500030000000703000000080a5461726765744c6561660000000600030000000703000000080a5461726765744c6561660000000700030000000703000000080a5461726765744c6561660000000800030000000703000000080a5461726765744c6561660000000900030000000703000000080a546172676574526f6f7400000002000a546172676574526f6f740000000300"), finalBytes);
        
        Assert.assertEquals(1, TargetRoot.root.counter);
        TargetArray checkArray = (TargetArray)TargetRoot.root;
        Assert.assertEquals(6, checkArray.array.length);
        Assert.assertEquals(9, ((TargetRoot)checkArray.array[5]).counter);
        Assert.assertTrue(((TargetLeaf)checkArray.array[5]).left == ((TargetLeaf)checkArray.array[0]).left);
        Assert.assertTrue(((TargetLeaf)checkArray.array[5]).right == ((TargetLeaf)checkArray.array[0]).right);
    }

    @Test
    public void testWithConstant() throws Exception {
        TargetRoot.root = new TargetRoot();
        TargetRoot.root.counter = 1;
        TargetLeaf next = new TargetLeaf();
        next.counter = 2;
        next.left = TEST_CONSTANT;
        next.right = next;
        TargetRoot.root.next = next;
        
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetRoot.class, TargetLeaf.class, TargetArray.class};
        byte[] finalBytes = serializeDeserializeAsNew(nextHashCode, sortedRoots);
        Assert.assertArrayEquals(Helpers.hexStringToBytes("00000001030000000000000000000000000a546172676574526f6f740000000103000000010a5461726765744c656166000000020002000000010300000001"), finalBytes);
        
        Assert.assertEquals(1, TargetRoot.root.counter);
        TargetLeaf checkNext = (TargetLeaf) TargetRoot.root.next;
        Assert.assertEquals(2, checkNext.counter);
        Assert.assertTrue(TEST_CONSTANT == checkNext.left);
        Assert.assertTrue(checkNext == checkNext.right);
    }

    @Test
    public void testWithClass() throws Exception {
        TargetArray array = new TargetArray(1);
        TargetRoot.root = array;
        TargetRoot.root.counter = 1;
        array.array[0] = TargetArray.class;
        
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetRoot.class, TargetLeaf.class, TargetArray.class};
        byte[] finalBytes = serializeDeserializeAsNew(nextHashCode, sortedRoots);
        Assert.assertArrayEquals(Helpers.hexStringToBytes("00000001030000000000000000000000000b5461726765744172726179000000010000000001010b5461726765744172726179"), finalBytes);
        
        TargetArray checkArray = (TargetArray) TargetRoot.root;
        Assert.assertEquals(1, checkArray.counter);
        Assert.assertTrue(TargetArray.class == checkArray.array[0]);
    }

    @Test
    public void testReentrantExample() throws Exception {
        TestGlobalResolver resolver = new TestGlobalResolver();
        TestNameMapper classNameMapper = new TestNameMapper();
        
        // We want to use a basic shared object shape, but then invoke the reentrant call to add a new object into the graph, earlier on (forcing the remapping logic to do something).
        TargetRoot.root = new TargetRoot();
        TargetRoot.root.counter = 1;
        TargetLeaf next = new TargetLeaf();
        next.counter = 2;
        next.left = next;
        next.right = next;
        TargetRoot.root.next = next;
        
        // We want to capture this state as the caller.
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetRoot.class, TargetLeaf.class};
        ReentrantGraph callerState = ReentrantGraph.captureCallerState(resolver, this.cache, classNameMapper, 1000, nextHashCode, sortedRoots, EmptyConstantClass.class);
        
        // We need to fake up a callee context, which means that shared instances will have a readIndex, so we need to create our new instances.
        TargetRoot.root = null;
        TargetLeaf.D = 0.0;
        int calleeHashCode = callerState.applyToRootsForNewFrame(resolver, this.cache, classNameMapper, sortedRoots, EmptyConstantClass.class);
        Assert.assertEquals(nextHashCode, calleeHashCode);
        
        // We want to now put these objects out of order, but keep as many as possible.  This means we add 1 single instance near the beginning.
        // Now, modify this shape and verify that we can write it back.
        TargetRoot newRoot = new TargetRoot();
        newRoot.counter = 3;
        newRoot.next = TargetRoot.root;
        TargetRoot.root = newRoot;
        TargetLeaf.D = 5.0;
        ReentrantGraph calleeState = ReentrantGraph.captureCalleeState(resolver, this.cache, classNameMapper, 1000, nextHashCode, sortedRoots, EmptyConstantClass.class);
        
        TargetRoot.root = null;
        TargetLeaf.D = 0.0;
        
        int hashCode = callerState.commitChangesToState(resolver, this.cache, classNameMapper, sortedRoots, EmptyConstantClass.class, calleeState);
        
        Assert.assertEquals(1, hashCode);
        Assert.assertEquals(3, TargetRoot.root.counter);
        TargetRoot extraRoot = TargetRoot.root.next;
        TargetLeaf checkNext = (TargetLeaf) extraRoot.next;
        Assert.assertEquals(2, checkNext.counter);
        Assert.assertTrue(checkNext == checkNext.left);
        Assert.assertTrue(checkNext == checkNext.right);
        // Make sure that the existing instance the test was still holding is still instance-equal to what is in the graph.
        Assert.assertTrue(checkNext == next);
    }

    @Test
    public void testReentrantRevert() throws Exception {
        TestGlobalResolver resolver = new TestGlobalResolver();
        TestNameMapper classNameMapper = new TestNameMapper();
        
        // We want to use a basic shared object shape, but then invoke the reentrant call to add a new object into the graph, earlier on (forcing the remapping logic to do something).
        TargetRoot.root = new TargetRoot();
        TargetRoot.root.counter = 1;
        TargetLeaf next = new TargetLeaf();
        next.counter = 2;
        next.left = next;
        next.right = next;
        TargetRoot.root.next = next;
        
        // We want to capture this state as the caller.
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetRoot.class, TargetLeaf.class};
        ReentrantGraph callerState = ReentrantGraph.captureCallerState(resolver, this.cache, classNameMapper, 1000, nextHashCode, sortedRoots, EmptyConstantClass.class);
        
        // We need to fake up a callee context, which means that shared instances will have a readIndex, so we need to create our new instances.
        TargetRoot.root = null;
        TargetLeaf.D = 0.0;
        int calleeHashCode = callerState.applyToRootsForNewFrame(resolver, this.cache, classNameMapper, sortedRoots, EmptyConstantClass.class);
        Assert.assertEquals(nextHashCode, calleeHashCode);
        
        // We want to now put these objects out of order, but keep as many as possible.  This means we add 1 single instance near the beginning.
        // Now, modify this shape and verify that we can write it back.
        TargetRoot newRoot = new TargetRoot();
        newRoot.counter = 3;
        newRoot.next = TargetRoot.root;
        TargetRoot.root = newRoot;
        TargetLeaf.D = 5.0;
        
        // Assume that, at this point, we decide to revert the changes.
        int hashCode = callerState.revertChangesToState(resolver, this.cache, classNameMapper, sortedRoots, EmptyConstantClass.class);
        
        Assert.assertEquals(1, hashCode);
        Assert.assertEquals(1, TargetRoot.root.counter);
        TargetLeaf checkNext = (TargetLeaf) TargetRoot.root.next;
        Assert.assertEquals(2, checkNext.counter);
        Assert.assertTrue(checkNext == checkNext.left);
        Assert.assertTrue(checkNext == checkNext.right);
        // Make sure that the existing instance the test was still holding is still instance-equal to what is in the graph.
        Assert.assertTrue(checkNext == next);
    }

    @Test
    public void testPerfIntArrays() throws Exception {
        int objectCount = 1000;
        int intCount = 1000;
        int samples = 10;
        TargetArray array = new TargetArray(objectCount);
        TargetRoot.root = array;
        
        for (int i = 0; i < objectCount; ++i) {
            TargetIntArray intArray = new TargetIntArray(intCount);
            for (int j = 0; j < intCount; ++j) {
                intArray.array[j] = j;
            }
            array.array[i] = intArray;
        }
        
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetRoot.class, TargetLeaf.class, TargetArray.class};
        byte[] finalBytes = serializeDeserializeAsNew(nextHashCode, sortedRoots);
        report("IntArrays perf serialized size: " + finalBytes.length);
        
        TestGlobalResolver resolver = new TestGlobalResolver();
        TestNameMapper classNameMapper = new TestNameMapper();
        
        // Do the serialization.
        ByteBuffer serializationBuffer = ByteBuffer.allocate(5_000_000);
        long start = System.nanoTime();
        for (int i = 0; i < samples; ++i) {
            serializationBuffer.clear();
            Serializer.serializeEntireGraph(serializationBuffer, null, null, resolver, this.cache, classNameMapper, nextHashCode, sortedRoots, EmptyConstantClass.class);
        }
        long end = System.nanoTime();
        long deltaNanosPer = (end - start) / samples;
        report("Serialized in " + deltaNanosPer + " ns");
        
        // Do the deserialization.
        ByteBuffer deserializationBuffer = ByteBuffer.wrap(finalBytes);
        start = System.nanoTime();
        for (int i = 0; i < samples; ++i) {
            deserializationBuffer.clear();
            Deserializer.deserializeEntireGraphAndNextHashCode(deserializationBuffer, null, resolver, this.cache, classNameMapper, sortedRoots, EmptyConstantClass.class);
        }
        end = System.nanoTime();
        deltaNanosPer = (end - start) / samples;
        report("Deserialized in " + deltaNanosPer + " ns");
    }

    @Test
    public void testPerfObjectArrays() throws Exception {
        int objectCount = 1000;
        int subCount = 100;
        int samples = 10;
        TargetArray array = new TargetArray(objectCount);
        TargetRoot.root = array;
        
        for (int i = 0; i < objectCount; ++i) {
            TargetArray subArray = new TargetArray(subCount);
            for (int j = 0; j < subCount; ++j) {
                subArray.array[j] = new TargetLeaf();
            }
            array.array[i] = subArray;
        }
        
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetRoot.class, TargetLeaf.class, TargetArray.class};
        byte[] finalBytes = serializeDeserializeAsNew(nextHashCode, sortedRoots);
        report("ObjectArrays perf serialized size: " + finalBytes.length);
        
        TestGlobalResolver resolver = new TestGlobalResolver();
        TestNameMapper classNameMapper = new TestNameMapper();
        
        // Do the serialization.
        ByteBuffer serializationBuffer = ByteBuffer.allocate(5_000_000);
        long start = System.nanoTime();
        for (int i = 0; i < samples; ++i) {
            serializationBuffer.clear();
            Serializer.serializeEntireGraph(serializationBuffer, null, null, resolver, this.cache, classNameMapper, nextHashCode, sortedRoots, EmptyConstantClass.class);
        }
        long end = System.nanoTime();
        long deltaNanosPer = (end - start) / samples;
        report("Serialized in " + deltaNanosPer + " ns");
        
        // Do the deserialization.
        ByteBuffer deserializationBuffer = ByteBuffer.wrap(finalBytes);
        start = System.nanoTime();
        for (int i = 0; i < samples; ++i) {
            deserializationBuffer.clear();
            Deserializer.deserializeEntireGraphAndNextHashCode(deserializationBuffer, null, resolver, this.cache, classNameMapper, sortedRoots, EmptyConstantClass.class);
        }
        end = System.nanoTime();
        deltaNanosPer = (end - start) / samples;
        report("Deserialized in " + deltaNanosPer + " ns");
    }

    @Test
    public void TestCleanClassStatics() throws Exception {
        int nextHashCode = 1;
        Class<?>[] sortedRoots = new Class<?>[] {TargetStatics.class};

        ByteBuffer buffer = ByteBuffer.allocate(1_000);
        TestGlobalResolver resolver = new TestGlobalResolver();
        TestNameMapper classNameMapper = new TestNameMapper();

        Serializer.serializeEntireGraph(buffer, null, null, resolver, this.cache, classNameMapper, nextHashCode, sortedRoots, EmptyConstantClass.class);

        Assert.assertTrue(null != sortedRoots[0].getDeclaredField("left").get(null));
        Assert.assertTrue(null != sortedRoots[0].getDeclaredField("right").get(null));

        Deserializer.cleanClassStatics(this.cache, sortedRoots, EmptyConstantClass.class);

        Assert.assertTrue(null == sortedRoots[0].getDeclaredField("left").get(null));
        Assert.assertTrue(null == sortedRoots[0].getDeclaredField("right").get(null));
    }


    private byte[] serializeDeserializeAsNew(int nextHashCode, Class<?>[] sortedRoots) {
        ByteBuffer buffer = ByteBuffer.allocate(5_000_000);
        TestGlobalResolver resolver = new TestGlobalResolver();
        TestNameMapper classNameMapper = new TestNameMapper();
        
        Serializer.serializeEntireGraph(buffer, null, null, resolver, this.cache, classNameMapper, nextHashCode, sortedRoots, EmptyConstantClass.class);
        byte[] finalBytes = new byte[buffer.position()];
        System.arraycopy(buffer.array(), 0, finalBytes, 0, finalBytes.length);
        
        ByteBuffer readingBuffer = ByteBuffer.wrap(finalBytes);
        int hashCode = Deserializer.deserializeEntireGraphAndNextHashCode(readingBuffer, null, resolver, this.cache, classNameMapper, sortedRoots, EmptyConstantClass.class);
        Assert.assertEquals(nextHashCode, hashCode);
        return finalBytes;
    }


    private static final class TestGlobalResolver implements IGlobalResolver {
        @Override
        public String getAsInternalClassName(Object target) {
            return (target instanceof Class)
                    ? ((Class<?>)target).getName()
                    : null;
        }
        @Override
        public int getAsConstant(Object target) {
            return (TEST_CONSTANT == target)
                    ? 1
                    : 0;
        }
        @Override
        public Object getClassObjectForInternalName(String internalClassName) {
            try {
                return Class.forName(internalClassName);
            } catch (ClassNotFoundException e) {
                // We can't fail to find this - we defined it.
                throw RuntimeAssertionError.unexpected(e);
            }
        }
        @Override
        public Object getConstantForIdentifier(int constantIdentifier) {
            return (1 == constantIdentifier)
                    ? TEST_CONSTANT
                    : null;
        }
    }

    private static final class TestNameMapper implements IPersistenceNameMapper {
        @Override
        public String getStorageClassName(String ourName) {
            // For testing purposes, we will say that the name mapping is to the simple name.
            return ourName.substring("org.aion.avm.core.persistence.".length());
        }
        @Override
        public String getInternalClassName(String storageClassName) {
            return "org.aion.avm.core.persistence." + storageClassName;
        }
    }

    private static final class EmptyConstantClass {
    }

    private static void report(String output) {
        if (REPORT) {
            System.out.println(output);
        }
    }
}
