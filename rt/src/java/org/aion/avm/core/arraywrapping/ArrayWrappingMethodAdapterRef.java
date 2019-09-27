package org.aion.avm.core.arraywrapping;

import a.BooleanArray;
import a.ByteArray;
import org.aion.avm.core.rejection.RejectedClassException;
import org.aion.avm.core.types.ClassHierarchy;
import org.aion.avm.core.util.Helpers;
import i.IObjectArray;
import i.RuntimeAssertionError;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;
import org.objectweb.asm.tree.AbstractInsnNode;
import org.objectweb.asm.tree.MethodInsnNode;
import org.objectweb.asm.tree.MethodNode;
import org.objectweb.asm.tree.TypeInsnNode;
import org.objectweb.asm.tree.analysis.Analyzer;
import org.objectweb.asm.tree.analysis.AnalyzerException;
import org.objectweb.asm.tree.analysis.BasicValue;
import org.objectweb.asm.tree.analysis.Frame;

/**
 * A method visitor that replace access bytecode
 *
 * AALOAD
 * AASTORE
 * BALOAD
 * BASTORE
 *
 * with corresponding array wrapper virtual call.
 *
 * Static analysis is required with {@link org.aion.avm.core.arraywrapping.ArrayWrappingInterpreter} so it can perform
 * type inference.
 *
 */

class ArrayWrappingMethodAdapterRef extends MethodNode implements Opcodes {
    private final ClassHierarchy hierarchy;

    private String className;
    private MethodVisitor mv;

    public ArrayWrappingMethodAdapterRef(final int access,
                                         final String name,
                                         final String descriptor,
                                         final String signature,
                                         final String[] exceptions,
                                         MethodVisitor mv,
                                         String className,
                                         ClassHierarchy hierarchy)
    {
        super(Opcodes.ASM6, access, name, descriptor, signature, exceptions);
        this.className = className;
        this.mv = mv;
        this.hierarchy = hierarchy;
    }

    @Override
    public void visitEnd(){

        Frame<BasicValue>[] frames = null;
        if (instructions.size() > 0) {
            try{
                Analyzer<BasicValue> analyzer = new Analyzer<>(new ArrayWrappingInterpreter(this.hierarchy));
                analyzer.analyze(this.className, this);
                frames = analyzer.getFrames();
            }catch (AnalyzerException e){
                // If we fail to run the analyzer, that is a serious internal error. It might be an actual bug
                // in the AVM, or it might be the result of corrupt input.
                // Since we're not sure, we "blame" the contract, and throw a Rejection Error.
                throw new RejectedClassException("Something went wrong when trying to analyze a wrapped array: " + e.getMessage());
            }
        }

        AbstractInsnNode[] insns = instructions.toArray();

        if (null != insns && null != frames) {
            RuntimeAssertionError.assertTrue(insns.length == frames.length);
        }

        for(int i = 0; i < insns.length; i++) {
            AbstractInsnNode insn = insns[i];
            Frame<BasicValue> f = frames[i];

            // We only handle aaload here since aastore is generic
            // the log is the following
            // check instruction -> check stack map frame -> replace instruction with invokeV and checkcast
            if (insn.getOpcode() == Opcodes.AALOAD) {
                //we pop the second slot on stack
                f.pop();
                BasicValue t = (BasicValue) (f.pop());
                String targetDesc = t.toString();
                String elementType = ArrayNameMapper.getElementType(targetDesc);

                MethodInsnNode invokeVNode =
                    new MethodInsnNode(Opcodes.INVOKEINTERFACE,
                                        Helpers.fulllyQualifiedNameToInternalName(IObjectArray.class.getName()),
                                        "get",
                                        "(I)Ljava/lang/Object;",
                                        true);

                TypeInsnNode checkcastNode =
                    new TypeInsnNode(Opcodes.CHECKCAST,
                                        elementType);

                // Insert indicate reverse order, we want
                // invokevirtual -> checkcast here
                instructions.insert(insn, checkcastNode);
                instructions.insert(insn, invokeVNode);
                instructions.remove(insn);
            }

            if (insn.getOpcode() == Opcodes.AASTORE) {
                //we pop the third slot on stack
                f.pop();
                f.pop();
                BasicValue t = (BasicValue) (f.pop());
                String targetDesc = t.toString();
                String elementType = ArrayNameMapper.getElementType(targetDesc);

                MethodInsnNode invokeVNode =
                        new MethodInsnNode(Opcodes.INVOKEINTERFACE,
                                Helpers.fulllyQualifiedNameToInternalName(IObjectArray.class.getName()),
                                "set",
                                "(ILjava/lang/Object;)V",
                                true);

                TypeInsnNode checkcastNode =
                        new TypeInsnNode(Opcodes.CHECKCAST,
                                elementType);

                // Insert indicate reverse order, we want
                // checkcast -> invokevirtual here
                instructions.insert(insn, invokeVNode);
                instructions.insert(insn, checkcastNode);
                instructions.remove(insn);
            }

            if (insn.getOpcode() == Opcodes.BALOAD) {
                f.pop();
                BasicValue t = f.pop();
                String targetDesc = t.toString();

                MethodInsnNode invokeVNode;
                if (targetDesc.equals("[Z")) {
                        invokeVNode = new MethodInsnNode(Opcodes.INVOKEVIRTUAL,
                                        Helpers.fulllyQualifiedNameToInternalName(BooleanArray.class.getName()),
                                        "get",
                                        "(I)Z",
                                        false);
                } else {
                        invokeVNode = new MethodInsnNode(Opcodes.INVOKEVIRTUAL,
                                        Helpers.fulllyQualifiedNameToInternalName(ByteArray.class.getName()),
                                        "get",
                                        "(I)B",
                                        false);
                }

                instructions.insert(insn, invokeVNode);
                instructions.remove(insn);
            }

            if (insn.getOpcode() == Opcodes.BASTORE) {
                f.pop();
                f.pop();
                BasicValue t = f.pop();
                String targetDesc = t.toString();

                MethodInsnNode invokeVNode;
                if (targetDesc.equals("[Z")) {
                        invokeVNode = new MethodInsnNode(Opcodes.INVOKEVIRTUAL,
                                        Helpers.fulllyQualifiedNameToInternalName(BooleanArray.class.getName()),
                                        "set",
                                        "(IZ)V",
                                        false);
                    } else {
                        invokeVNode = new MethodInsnNode(Opcodes.INVOKEVIRTUAL,
                                        Helpers.fulllyQualifiedNameToInternalName(ByteArray.class.getName()),
                                        "set",
                                        "(IB)V",
                                        false);
                    }
                instructions.insert(insn, invokeVNode);
                instructions.remove(insn);
            }
        }

        accept(mv);
    }
}
