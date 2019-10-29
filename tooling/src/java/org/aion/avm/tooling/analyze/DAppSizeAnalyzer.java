package org.aion.avm.tooling.analyze;

import java.io.ByteArrayInputStream;
import java.io.FileInputStream;
import java.io.IOException;
import java.util.Map;
import java.util.jar.JarInputStream;

import org.aion.avm.utilities.Utilities;
import org.aion.avm.utilities.analyze.ClassFileInfoBuilder;

public class DAppSizeAnalyzer {

    public static void main(String[] args) {
        if (args.length != 1) {
            System.err.println("Input the path to the jar file.");
            System.exit(0);
        }

        try (FileInputStream fileInputStream = new FileInputStream(args[0])) {
            analyze(fileInputStream.readAllBytes());

        } catch (IOException e) {
            e.printStackTrace();
            System.exit(0);
        }
    }

    private static void analyze(byte[] jarBytes) {

        Map<String, byte[]> classMap;
        try {
            JarInputStream jarReader = new JarInputStream(new ByteArrayInputStream(jarBytes), true);
            classMap = Utilities.extractClasses(jarReader, Utilities.NameStyle.SLASH_NAME);
            for (Map.Entry<String, byte[]> classEntry : classMap.entrySet()) {
                ClassFileInfoBuilder.ClassFileInfo info = ClassFileInfoBuilder.getClassFileInfo(classEntry.getValue());
                printResult(classEntry.getKey(), info);
            }

        } catch (IOException e) {
            e.printStackTrace();
            throw new RuntimeException(e);
        }
    }

    private static void printResult(String className, ClassFileInfoBuilder.ClassFileInfo info) {
        System.out.format("*****************************************************************************%n");

        System.out.format("%-20s %-70s %n", "Class Name:", className);
        System.out.format("%-20s %-70s %n %n", "Class File Length:", info.classFileLength);

        System.out.format("%-20s %-70s %n", "Instance Field Count: ", info.instanceFieldCount);
        System.out.format("%-20s %-70s %n %n", "Defined Method Count: ", info.definedMethods.size());

        System.out.format("%-20s %-70s %n", "Constant Pool Entry Count: ", info.constantPoolEntryCount);
        System.out.format("%-20s %-70s %n", "Constant Pool Byte Size: ", info.totalConstantPoolByteSize);
        System.out.format("%-20s %-70s %n %n", "Total Utf8 Byte Length: ", info.totalUtf8ByteLength);

        System.out.format("-----------------------------%n");
        System.out.format("%-20s | %-10s %n", "Constant Type", "Count");
        System.out.format("-----------------------------%n");

        for (Map.Entry<String, Integer> constantType : info.constantTypeCount.entrySet()) {
            System.out.format("%-20s | %-10s %n", constantType.getKey(), constantType.getValue());
        }
        System.out.format("-----------------------------%n%n");
    }
}
