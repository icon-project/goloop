package org.aion.avm.core.persistence;


public class TargetLeaf extends TargetRoot {
    public static double D;
    public TargetRoot left;
    public TargetRoot right;
    public TargetLeaf() {
    }
    public TargetLeaf(Void ignore, int readIndex) {
        super(ignore, readIndex);
    }
}
