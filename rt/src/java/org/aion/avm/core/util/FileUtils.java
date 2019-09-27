package org.aion.avm.core.util;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.nio.file.FileSystems;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;

/**
 * @author Roman Katerinenko
 */
public class FileUtils {
    public static Path putToTempDir(byte[] bytes, String newDirName, String newFileName) throws IOException {
        Path dstJarPath = Files.createTempDirectory(newDirName)
                .resolve(newFileName)
                .toAbsolutePath()
                .normalize();
        try (final InputStream in = new ByteArrayInputStream(bytes)) {
            Files.copy(in, dstJarPath, StandardCopyOption.REPLACE_EXISTING);
        }
        return dstJarPath;
    }

    public static Path getFSRootDirFor(Path pathToJar) throws IOException {
        final var fileSystem = FileSystems.newFileSystem(pathToJar, null);
        return fileSystem.getRootDirectories().iterator().next();
    }
}