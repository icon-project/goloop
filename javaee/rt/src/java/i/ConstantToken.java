package i;

public class ConstantToken {

    public final int constantId;

    public ConstantToken(int constantId) {
        this.constantId = constantId;
    }

    public static int getReadIndexFromConstantId(int constantId){
        return (-1 * constantId) - 1;
    }

    public static int getConstantIdFromReadIndex(int readIndex){
        return (-1 * readIndex) - 1;
    }
}
