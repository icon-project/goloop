package a;

public enum ArrayElement {

    // Integral type
    BYTE(1),
    SHORT(2),
    INT(4),
    LONG(8),
    CHAR(2),

    // Floating-point type
    FLOAT(4),
    DOUBLE(8),

    // Reference type
    REF(8);

    static final int COST_PER_BYTE = 3;

    private final int energy;

    ArrayElement(int size) {
        this.energy = size * COST_PER_BYTE;
    }

    public int getEnergy() {
        return energy;
    }
}
