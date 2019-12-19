/*
 * Copyright 2019 ICON Foundation
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

import foundation.icon.ee.tooling.deploy.OptimizedJarBuilder;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

public class DAppCompiler {
    private static boolean DEBUG_MODE = false;

    public static void main(String[] args) throws IOException {
        Logger logger = LoggerFactory.getLogger(DAppCompiler.class);
        logger.info("=== DAppCompiler ===");
        if (args.length > 0 && args.length <= 2) {
            if (args.length == 2) {
                DEBUG_MODE = "-debug".equals(args[1]);
            }
            OptimizedJarBuilder jarBuilder = new OptimizedJarBuilder(DEBUG_MODE, readFile(args[0]))
                    .withUnreachableMethodRemover()
                    .withRenamer();
            byte[] optimizedJar = jarBuilder.getOptimizedBytes();
            String outputName = getJarFilename(args[0], DEBUG_MODE);
            writeFile(outputName, optimizedJar);
            logger.info("Generated {}", outputName);
        } else {
            logger.info("Usage: DAppCompiler <jarFile> (-debug)");
        }
    }

    private static String getJarFilename(String input, boolean debugMode) {
        int len = input.lastIndexOf("/") + 1;
        String prefix = input.substring(0, len) + "optimized";
        if (debugMode) {
            return prefix + "-debug.jar";
        } else {
            return prefix + ".jar";
        }
    }

    private static byte[] readFile(String jarFile) throws IOException {
        Path path = Paths.get(jarFile);
        byte[] jarBytes;
        try {
            jarBytes = Files.readAllBytes(path);
        } catch (IOException e) {
            throw new IOException("JAR read error: " + e.getMessage());
        }
        return jarBytes;
    }

    private static void writeFile(String filePath, byte[] data) {
        Path outFile = Paths.get(filePath);
        try {
            Files.write(outFile, data);
        } catch (IOException e) {
            throw new RuntimeException(e.getMessage());
        }
    }
}
