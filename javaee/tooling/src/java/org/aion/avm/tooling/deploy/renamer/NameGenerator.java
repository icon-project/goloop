package org.aion.avm.tooling.deploy.renamer;

import java.util.Set;

public class NameGenerator {
    private static final char[] CHARACTERS = new char[26];

    static {
        for (int i = 'a'; i <= 'z'; i++) {
            CHARACTERS[i - 'a'] = ((char) i);
        }
    }

    private int currentClassIndex;
    private int currentInstructionIndex;

    public NameGenerator() {
        currentClassIndex = 0;
        currentInstructionIndex = 0;
    }

    private static String nextString(int i) {
        return i < 0 ? "" : nextString((i / 26) - 1) + CHARACTERS[i % 26];
    }

    public String getNextClassName() {
        String className = nextString(currentClassIndex);
        currentClassIndex++;
        return className.toUpperCase();
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
}
