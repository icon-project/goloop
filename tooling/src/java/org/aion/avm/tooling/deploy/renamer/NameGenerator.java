package org.aion.avm.tooling.deploy.renamer;

import java.util.Set;

public class NameGenerator {
    private static final char[] CHARACTERS = new char[26];
    static {
        for (int i = 97; i <= 122; i++) {
            CHARACTERS[i - 97] = ((char) i);
        }
    }

    private int currentClassIndex;
    private int currentInstructionIndex;

    public NameGenerator() {
        currentClassIndex = 1;
    }

    public String getNextClassName() {
        String className = nextString(currentClassIndex).toUpperCase();
        currentClassIndex++;
        return className.toUpperCase();
    }

    // main class will always be mapped to A
    public static String getNewMainClassName() {
        return String.valueOf(CHARACTERS[0]).toUpperCase();
    }

    public String getNextMethodOrFieldName(Set<String> restrictions) {
        String name = nextString(currentInstructionIndex);
        if (restrictions != null) {
            while (restrictions.contains(name)) {
                currentInstructionIndex++;
                name = nextString(currentInstructionIndex);
            }
        }
        currentInstructionIndex++;
        return name;
    }

    private static String nextString(int i) {
        return i < 0 ? "" : nextString((i / 26) - 1) + CHARACTERS[i % 26];
    }
}
