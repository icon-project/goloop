package s.java.lang;

import foundation.icon.ee.util.LogMarker;
import i.IInstrumentation;
import i.IObjectDeserializer;
import i.IObjectSerializer;
import org.aion.avm.ClassNameExtractor;
import org.aion.avm.RuntimeMethodFeeSchedule;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import s.java.io.Serializable;

import java.util.Collections;
import java.util.IdentityHashMap;
import java.util.Set;

/**
 * Our shadow implementation of java.lang.Throwable.
 * 
 * NOTE:  Instances of this class never actually touch the underlying VM-generated exception.
 * If we want to carry that information around, we will need a new constructor, an addition to the generated stubs, and a sense of how to use it.
 * Avoiding carrying those instances around means that this implementation becomes very safely defined.
 * It does, however, mean that we can't expose stack traces since those are part of the VM-generated exceptions.
 *
 * NOTE: All shadow Throwable and its derived exceptions and errors' APIs are not billed; since the native exception object is not billed in the constructor,
 * and we replace them with the shadow instances only when it is caught (in a catch or finally block), to have a more consistent fee schedule, the shadow
 * methods are free of energy charges as well. Then the user doesn't experience different charges in slightly different scenarios (created and thrown, caught or not caught).
 * Also note that at the creation of these exception/error objects, the 'new' bytecode and the heap size are billed.
 */
public class Throwable extends Object implements Serializable {
    static {
        // Shadow classes MUST be loaded during bootstrap phase.
        IInstrumentation.attachedThreadInstrumentation.get().bootstrapOnly();
    }

    private String message;
    private Throwable cause;
    private java.lang.String systemMessage = null;
    private StackTraceElement[] stackTrace;

    public Throwable() {
        this((String)null, (Throwable)null);
    }

    public Throwable(String message) {
        this(message, null);
    }

    public Throwable(String message, Throwable cause) {
        this.message = message;
        this.cause = cause;
        this.stackTrace = Thread.currentThread().getStackTrace();
    }

    public Throwable(Throwable cause) {
        this.message = (cause == null ? null : cause.internalToString());
        this.cause = cause;
        this.stackTrace = Thread.currentThread().getStackTrace();
    }

    // Deserializer support.
    public Throwable(java.lang.Void ignore, int readIndex) {
        super(ignore, readIndex);
    }

    public void deserializeSelf(java.lang.Class<?> firstRealImplementation, IObjectDeserializer deserializer) {
        super.deserializeSelf(Throwable.class, deserializer);
        this.message = (String) deserializer.readObject();
        this.cause = (Throwable) deserializer.readObject();
    }

    public void serializeSelf(java.lang.Class<?> firstRealImplementation, IObjectSerializer serializer) {
        super.serializeSelf(Throwable.class, serializer);
        serializer.writeObject(message);
        serializer.writeObject(cause);
        // DO NOT serialize systemMessage and backtrace as this can change
        // between versions
    }

    public String avm_getMessage() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Throwable_Hierarchy_Base_Fee);
        lazyLoad();
        return this.message;
    }

    public String avm_getLocalizedMessage() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Throwable_Hierarchy_Base_Fee);
        lazyLoad();
        return this.message;
    }

    public Throwable avm_getCause() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Throwable_Hierarchy_Base_Fee);
        lazyLoad();
        return this.cause;
    }

    public Throwable avm_initCause(Throwable cause) {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Throwable_Hierarchy_Base_Fee);
        lazyLoad();
        this.cause = cause;
        return this;
    }

    public String avm_toString() {
        IInstrumentation.attachedThreadInstrumentation.get().chargeEnergy(RuntimeMethodFeeSchedule.Throwable_avm_toString);
        lazyLoad();
        return internalToString();
    }

    public void avm_printStackTrace() {
        Set<Throwable> visited = Collections.newSetFromMap(new IdentityHashMap<>());
        var sb = new java.lang.StringBuilder();
        buildStackTraceString(sb, "", visited);
        final Logger logger = LoggerFactory.getLogger(Throwable.class);
        logger.trace(LogMarker.Trace, "PRT| {}", sb);
    }

    private void buildStackTraceString(java.lang.StringBuilder sb,
            java.lang.String caption,
            Set<Throwable> visited) {
        if (visited.contains(this)) {
            return;
        }
        visited.add(this);
        sb.append(caption).append(this).append("\n");
        if (stackTrace != null) {
            for (StackTraceElement e : stackTrace) {
                sb.append("\tat ").append(e).append("\n");
            }
        }

        if (cause != null) {
            cause.buildStackTraceString(sb, "Caused by: ", visited);
        }
    }

    public void setSystemMessage(java.lang.String message) {
        this.systemMessage = message;
    }

    public void setStackTrace(StackTraceElement[] backtrace) {
        this.stackTrace = backtrace;
    }

    public java.lang.String getMessage() {
        return this.message != null ? this.message.getUnderlying() : null;
    }

    //=======================================================
    // Methods below are used by runtime and test code only!
    //========================================================

    @Override
    public java.lang.String toString() {
        lazyLoad();
        return getClass().getName() + ": "
                + (this.message==null ? "" : this.message) + ": "
                + (this.systemMessage==null ? "" : this.systemMessage);
    }

    private String internalToString(){
        String s = new String(ClassNameExtractor.getOriginalClassName(getClass().getName()));
        return (this.message != null) ? new String(s + ": " + this.message) : s;
    }
}
