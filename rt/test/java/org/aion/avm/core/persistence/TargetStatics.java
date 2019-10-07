package org.aion.avm.core.persistence;


public class TargetStatics extends TargetRoot {
    public static TargetRoot left = new TargetLeaf();
    public static TargetRoot right = new TargetLeaf();

    public TargetStatics() {
    }
    public TargetStatics(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
