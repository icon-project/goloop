/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import org.aion.avm.core.util.AllowlistProvider;
import org.aion.avm.utilities.Utilities;

import java.util.Comparator;

public class JCLAllowlistPrinter {
    private static boolean printMethods = true;

    public static void main(String[] args) {
        try {
            if (args.length > 0 && "-no_methods".equals(args[0])) {
                printMethods = false;
            }
            var allowlist = AllowlistProvider.getClassLibraryMap();
            allowlist.keySet().stream()
                .sorted(Comparator.comparing(Class::getName))
                .forEach(clazz -> {
                    String className = Utilities.fullyQualifiedNameToInternalName(clazz.getName());
                    if (className.startsWith("java")) {
                        System.out.println(className);
                        if (printMethods) {
                            allowlist.get(clazz).stream()
                                .sorted(Comparator.comparing(AllowlistProvider.MethodDescriptor::toString))
                                .forEach(md -> {
                                    System.out.printf("  - %s%s %s\n",
                                            md.name, md.parameters, md.isStatic ? "(static)" : "");
                                });
                        }
                    }
                });
        } catch (ClassNotFoundException e) {
            e.printStackTrace();
        }
    }
}
