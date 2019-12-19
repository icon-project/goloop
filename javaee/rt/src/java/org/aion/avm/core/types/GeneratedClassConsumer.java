package org.aion.avm.core.types;

/**
 * An interface which exists to allow external concerns to be notified when a class is dynamically generated for the purposes of exception wrapping.
 */
public interface GeneratedClassConsumer {
    /**
     * Called when a new class is generated.
     *
     * @param superClassName The name of the super class (in slash form).
     * @param className The name of the generated class (in slash form).
     * @param bytecode The bytecode of the generated class.
     */
    public void accept(String superClassName, String className, byte[] bytecode);
}
