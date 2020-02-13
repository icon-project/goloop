/*
 * Copyright (c) 2019 ICON Foundation
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

package foundation.icon.test.common;

import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.crypto.KeystoreException;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.data.TransactionResult.EventLog;

import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.IOException;
import java.math.BigDecimal;
import java.math.BigInteger;
import java.nio.file.Files;
import java.util.List;
import java.util.zip.ZipEntry;
import java.util.zip.ZipOutputStream;

import static foundation.icon.test.common.Env.LOG;

public class Utils {
    public static KeyWallet readWalletFromFile(String path, String password) throws IOException {
        try {
            File file = new File(path);
            return KeyWallet.load(password, file);
        } catch (KeystoreException e) {
            e.printStackTrace();
            throw new IOException("Key load failed!");
        }
    }

    public static void ensureIcxBalance(TransactionHandler txHandler, Address address,
                                        long oldVal, long newVal) throws Exception {
        BigInteger oldValInt = BigInteger.valueOf(oldVal).multiply(BigDecimal.TEN.pow(18).toBigInteger());
        BigInteger newValInt = BigInteger.valueOf(newVal).multiply(BigDecimal.TEN.pow(18).toBigInteger());
        Utils.ensureIcxBalance(txHandler, address, oldValInt, newValInt);
    }

    public static void ensureIcxBalance(TransactionHandler txHandler, Address address,
                                        BigInteger oldVal, BigInteger newVal) throws Exception {
        long limitTime = System.currentTimeMillis() + Constants.DEFAULT_WAITING_TIME;
        while (true) {
            BigInteger icxBalance = txHandler.getBalance(address);
            String msg = "ICX balance of " + address + ": " + icxBalance;
            if (icxBalance.equals(oldVal)) {
                if (limitTime < System.currentTimeMillis()) {
                    throw new ResultTimeoutException();
                }
                try {
                    // wait until block confirmation
                    LOG.debug(msg + "; Retry in 1 sec.");
                    Thread.sleep(1000);
                } catch (InterruptedException e) {
                    e.printStackTrace();
                }
            } else if (icxBalance.equals(newVal)) {
                LOG.info(msg);
                break;
            } else {
                throw new IOException("ICX balance mismatch!");
            }
        }
    }

    public static EventLog findEventLogWithFuncSig(TransactionResult result, Address scoreAddress, String funcSig) {
        List<EventLog> eventLogs = result.getEventLogs();
        for (EventLog event : eventLogs) {
            if (event.getScoreAddress().equals(scoreAddress.toString())) {
                String signature = event.getIndexed().get(0).asString();
                LOG.debug("function sig: " + signature);
                if (funcSig.equals(signature)) {
                    return event;
                }
            }
        }
        return null ;
    }

    private static void recursiveZip(File source, String zipPath, ZipOutputStream zos) throws IOException{
        if(source.isHidden()) {
            return;
        }
        if(source.isDirectory()) {
            String dir = source.getName();
            if(!dir.endsWith(File.separator)) {
                dir = dir + File.separator;
            }
            zos.putNextEntry(new ZipEntry(dir));
            zos.closeEntry();
            File []files = source.listFiles();
            if(files == null) {
                return;
            }
            String path = zipPath == null ? dir : zipPath + dir;
            for(File file : files) {
                recursiveZip(file, path, zos);
            }
        }
        else {
            ZipEntry ze = new ZipEntry(zipPath + source.getName());
            zos.putNextEntry(ze);
            zos.write(Files.readAllBytes(source.toPath()));
            zos.closeEntry();
        }
    }

    public static byte[] zipContent(String path) throws IOException {
        ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
        ZipOutputStream zos = new ZipOutputStream(outputStream);
        recursiveZip(new File(path), null, zos);
        zos.close();
        outputStream.close();
        return outputStream.toByteArray();
    }
}
