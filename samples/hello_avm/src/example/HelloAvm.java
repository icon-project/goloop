package example;

import avm.Blockchain;
import org.aion.avm.tooling.abi.Callable;
import org.aion.avm.userlib.abi.ABIDecoder;

public class HelloAvm
{
    private static String myStr;

    static {
        Blockchain.println("***** HelloAvm <clinit> Start *****");
        ABIDecoder decoder = new ABIDecoder(Blockchain.getData());
        myStr = decoder.decodeOneString();
        Blockchain.println("*** myStr = " + myStr);
    }

    @Callable
    public static void sayHello() {
        Blockchain.println(myStr);
    }

    @Callable
    public static String greet(String name) {
        return "Hello " + name;
    }

    @Callable
    public static String getString() {
        Blockchain.println("Current string is " + myStr);
        return myStr;
    }

    @Callable
    public static void setString(String newStr) {
        myStr = newStr;
        Blockchain.println("New string is " + myStr);
    }
}
