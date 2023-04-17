/*
 * Copyright 2022 ICON Foundation
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

package testcases;

import score.Context;
import score.annotation.External;

import java.math.BigInteger;
import java.util.Arrays;

public class BLSTestScore {
    static private byte[] hexToBytes(String s) {
        int len = s.length();
        byte[] data = new byte[len / 2];
        for (int i = 0; i < len; i += 2) {
            data[i / 2] = (byte) ((Character.digit(s.charAt(i), 16) << 4)
                    + Character.digit(s.charAt(i + 1), 16));
        }
        return data;
    }
    static private final String G1 = "bls12-381-g1";
    static private final String G2 = "bls12-381-g2";
    static private final String pa = "a85840694564cd1582f53e30fca43a396214990e5e0b255b8d257931ff0a933a5746b3a9bdd63b9c93ade10a85db0e9b";
    static private final String pb = "b0f6fc69e358da7acefc579b5c87bd5970257a347fc45aa53e73d6c65fe5354ce63f25e27412d301ba7e4661b65175f3";
    static private final String pa_plus_pb ="ae8831c4f88dfb7853af6b0c4db9fd38becb236dfbe64633c782a2796544fb8e751edcd996b0b19826a0c33fee80805b";
    static private final String pMalformed = pa.substring(0, pa.length()-2);

    static private final String pk = "a931985bb2949bd7bebf453e6ca3b653d4c661d90316e5ec0d844f3c187c2920799c605e76ff64184d0e0f5c1f69e955";
    static private final String pkMalformed = pk.substring(0, pk.length()-2);
    static private final String msg = "6d79206d657373616765";
    static private final String sig = "a9d535044a303502a75c2364570731069f862858a1b0a60ae7c2981b4aa96fa48fe8c4a25d000a2a75b0653c60658dd00ebac42bcef4b9a6fc293dce6e207e10040c909b1f3d2be5ebf55f1865d6b66d72eb8d9379df0b2a737d01de84813af1";

    static private final String sigMalformed = sig.substring(0, sig.length()-2);
    static private final String msg2 = "6d79206d657373616766";

    @External
    public void test() {
        testAggregate0();
        testAggregate1();
        testAggregate2();
        testAggregateMalformed();
        testVerifySignature();
        testVerifySignatureMalformedPK();
        testVerifySignatureMalformedSig();
        testVerifySignatureNotMatch();

        testBLS12381ecAddG1();
        testBLS12381ecAddG1Compressed();
        testBLS12381ecScalarMulG1();
        testBLS12381ecScalarMulG1Compressed();
        testBLS12381ecAddG2();
        testBLS12381ecAddG2Compressed();
        testBLS12381ecScalarMulG2();
        testBLS12381ecScalarMulG2Compressed();
        testBLS12381ecPairingCheck();
        testBLS12381ecPairingCheckCompressed();
        testBLS12381InvalidDataEncoding();
    }

    public void testAggregate0() {
        var id = Context.aggregate(G1, null, new byte[0]);
        Context.require(Arrays.equals(
                id, Context.aggregate(G1, id, id)
        ));
        Context.println("testAggregate0 - OK");
    }

    public void testAggregate1() {
        Context.require(Arrays.equals(
                hexToBytes(pa),
                Context.aggregate(G1, hexToBytes(pa), new byte[0])
        ));
        Context.require(Arrays.equals(
                hexToBytes(pa),
                Context.aggregate(G1, null, hexToBytes(pa))
        ));
        Context.println("testAggregate1 - OK");
    }

    public void testAggregate2() {
        Context.require(Arrays.equals(
                hexToBytes(pa_plus_pb),
                Context.aggregate(G1, null, hexToBytes(pa+pb))
        ));
        var id = Context.aggregate(G1, null, new byte[0]);
        Context.require(Arrays.equals(
                hexToBytes(pa_plus_pb),
                Context.aggregate(G1, id, hexToBytes(pa+pb))
        ));
        Context.require(Arrays.equals(
                hexToBytes(pa_plus_pb),
                Context.aggregate(G1, hexToBytes(pa), hexToBytes(pb))
        ));
        Context.println("testAggregate2 - OK");
    }

    public void testAggregateMalformed() {
        try {
            Context.aggregate(G1, hexToBytes(pMalformed), hexToBytes(pa));
            Context.require(false, "shall not reach here");
        } catch (IllegalArgumentException e) {
        }
        Context.println("testAggregateMalformed - OK");
    }

    public void testVerifySignature() {
        Context.require(Context.verifySignature(G2, hexToBytes(msg), hexToBytes(sig), hexToBytes(pk)));
        Context.println("testVerifySignature - OK");
    }

    public void testVerifySignatureMalformedPK() {
        try {
            Context.verifySignature(G2, hexToBytes(msg), hexToBytes(sig), hexToBytes(pkMalformed));
            Context.require(false, "shall not reach here");
        } catch (IllegalArgumentException e) {
        }
        Context.println("testVerifySignatureMalformedPK - OK");
    }

    public void testVerifySignatureMalformedSig() {
        try {
            Context.verifySignature(G2, hexToBytes(msg), hexToBytes(sigMalformed), hexToBytes(pk));
            Context.require(false, "shall not reach here");
        } catch (IllegalArgumentException e) {
        }
        Context.println("testVerifySignatureMalformedSig - OK");
    }

    public void testVerifySignatureNotMatch() {
        Context.require(!Context.verifySignature(G2, hexToBytes(msg2), hexToBytes(sig), hexToBytes(pk)));
        Context.println("testVerifySignatureNotMatch - OK");
    }

    public void testBLS12381ecAddG1() {
        byte[] g1b = hexToBytes("17f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb08b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1");
        byte[] g1x2b = hexToBytes("0572cbea904d67468808c8eb50a9450c9721db309128012543902d0ac358a62ae28f75bb8f1c7c42c39a8c5529bf0f4e166a9d8cabc673a322fda673779d8e3822ba3ecb8670e461f73bb9021d5fd76a4c56d9d4cd16bd1bba86881979749d28");
        byte[] g1x3b = hexToBytes("09ece308f9d1f0131765212deca99697b112d61f9be9a5f1f3780a51335b3ff981747a0b2ca2179b96d2c0c9024e5224032b80d3a6f5b09f8a84623389c5f80ca69a0cddabc3097f9d9c27310fd43be6e745256c634af45ca3473b0590ae30d1");
        byte[] g1x6b = hexToBytes("06e82f6da4520f85c5d27d8f329eccfa05944fd1096b20734c894966d12a9e2a9a9744529d7212d33883113a0cadb90917d81038f7d60bee9110d9c0d6d1102fe2d998c957f28e31ec284cc04134df8e47e8f82ff3af2e60a6d9688a4563477c");

        byte[] out = Context.ecAdd("bls12-381-g1", concatBytes(g1b, g1x2b, g1x3b), false);
        Context.require(Arrays.equals(g1x6b, out), "incorrect ecAddG1 result");
        
        Context.println("testBLS12381ecAddG1 - OK");
    }

    public void testBLS12381ecAddG1Compressed() {
        byte[] g1b = hexToBytes("97f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb");
        byte[] g1x2b = hexToBytes("a572cbea904d67468808c8eb50a9450c9721db309128012543902d0ac358a62ae28f75bb8f1c7c42c39a8c5529bf0f4e");
        byte[] g1x3b = hexToBytes("89ece308f9d1f0131765212deca99697b112d61f9be9a5f1f3780a51335b3ff981747a0b2ca2179b96d2c0c9024e5224");
        byte[] g1x6b = hexToBytes("a6e82f6da4520f85c5d27d8f329eccfa05944fd1096b20734c894966d12a9e2a9a9744529d7212d33883113a0cadb909");

        byte[] out = Context.ecAdd("bls12-381-g1", concatBytes(g1b, g1x2b, g1x3b), true);
        Context.require(Arrays.equals(g1x6b, out), "incorrect ecAddG1 result");
        
        Context.println("testBLS12381ecAddG1Compressed - OK");
    }

    public void testBLS12381ecScalarMulG1() {
        byte[] g1b = hexToBytes("17f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb08b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1");
        byte[] g1x2b = hexToBytes("0572cbea904d67468808c8eb50a9450c9721db309128012543902d0ac358a62ae28f75bb8f1c7c42c39a8c5529bf0f4e166a9d8cabc673a322fda673779d8e3822ba3ecb8670e461f73bb9021d5fd76a4c56d9d4cd16bd1bba86881979749d28");
        byte[] g1x6b = hexToBytes("06e82f6da4520f85c5d27d8f329eccfa05944fd1096b20734c894966d12a9e2a9a9744529d7212d33883113a0cadb90917d81038f7d60bee9110d9c0d6d1102fe2d998c957f28e31ec284cc04134df8e47e8f82ff3af2e60a6d9688a4563477c");
       
        byte[] out;
        
        out = Context.ecScalarMul("bls12-381-g1", new BigInteger("2").toByteArray(), g1b, false);
        Context.require(Arrays.equals(g1x2b, out), "incorrect ecAdd result");
        
        out = Context.ecScalarMul("bls12-381-g1", new BigInteger("3").toByteArray(), g1x2b, false);
        Context.require(Arrays.equals(g1x6b, out), "incorrect ecScalarMulG1 result");

        Context.println("testBLS12381ecScalarMulG1 - OK");
    }

    public void testBLS12381ecScalarMulG1Compressed() {
        byte[] g1b = hexToBytes("97f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb");
        byte[] g1x2b = hexToBytes("a572cbea904d67468808c8eb50a9450c9721db309128012543902d0ac358a62ae28f75bb8f1c7c42c39a8c5529bf0f4e");
        byte[] g1x6b = hexToBytes("a6e82f6da4520f85c5d27d8f329eccfa05944fd1096b20734c894966d12a9e2a9a9744529d7212d33883113a0cadb909");
        
        byte[] out;
        
        out = Context.ecScalarMul("bls12-381-g1", new BigInteger("2").toByteArray(), g1b, true);
        Context.require(Arrays.equals(g1x2b, out), "incorrect ecScalarMul result");
        
        out = Context.ecScalarMul("bls12-381-g1", new BigInteger("3").toByteArray(), g1x2b, true);
        Context.require(Arrays.equals(g1x6b, out), "incorrect ecScalarMulG1 result");

        Context.println("testBLS12381ecScalarMulG1Compressed - OK");
    }

    public void testBLS12381ecAddG2() {
        byte[] g2b = hexToBytes("024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb813e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e0ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b828010606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be");
        byte[] g2x2b = hexToBytes("1638533957d540a9d2370f17cc7ed5863bc0b995b8825e0ee1ea1e1e4d00dbae81f14b0bf3611b78c952aacab827a0530a4edef9c1ed7f729f520e47730a124fd70662a904ba1074728114d1031e1572c6c886f6b57ec72a6178288c47c335770468fb440d82b0630aeb8dca2b5256789a66da69bf91009cbfe6bd221e47aa8ae88dece9764bf3bd999d95d71e4c98990f6d4552fa65dd2638b361543f887136a43253d9c66c411697003f7a13c308f5422e1aa0a59c8967acdefd8b6e36ccf3");
        byte[] g2x3b = hexToBytes("122915c824a0857e2ee414a3dccb23ae691ae54329781315a0c75df1c04d6d7a50a030fc866f09d516020ef82324afae09380275bbc8e5dcea7dc4dd7e0550ff2ac480905396eda55062650f8d251c96eb480673937cc6d9d6a44aaa56ca66dc0b21da7955969e61010c7a1abc1a6f0136961d1e3b20b1a7326ac738fef5c721479dfd948b52fdf2455e44813ecfd89208f239ba329b3967fe48d718a36cfe5f62a7e42e0bf1c1ed714150a166bfbd6bcf6b3b58b975b9edea56d53f23a0e849");
        byte[] g2x6b = hexToBytes("19e384121b7d70927c49e6d044fd8517c36bc6ed2813a8956dd64f049869e8a77f7e46930240e6984abe26fa6a89658f03f4b4e761936d90fd5f55f99087138a07a69755ad4a46e4dd1c2cfe6d11371e1cc033111a0595e3bba98d0f538db45117a31a4fccfb5f768a2157517c77a4f8aaf0dee8f260d96e02e1175a8754d09600923beae02a019afc327b65a2fdbbfc088bb5832f4a4a452edda646ebaa2853a54205d56329960b44b2450070734724a74daaa401879bad142132316e9b3401");

        byte[] out = Context.ecAdd("bls12-381-g2", concatBytes(g2b, g2x2b, g2x3b), false);
        Context.require(Arrays.equals(g2x6b, out), "incorrect ecAdd.G2 result");
        
        Context.println("testBLS12381ecAddG2 - OK");
    }

    public void testBLS12381ecAddG2Compressed() {
        byte[] g2b = hexToBytes("93e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8");
        byte[] g2x2b = hexToBytes("aa4edef9c1ed7f729f520e47730a124fd70662a904ba1074728114d1031e1572c6c886f6b57ec72a6178288c47c335771638533957d540a9d2370f17cc7ed5863bc0b995b8825e0ee1ea1e1e4d00dbae81f14b0bf3611b78c952aacab827a053");
        byte[] g2x3b = hexToBytes("89380275bbc8e5dcea7dc4dd7e0550ff2ac480905396eda55062650f8d251c96eb480673937cc6d9d6a44aaa56ca66dc122915c824a0857e2ee414a3dccb23ae691ae54329781315a0c75df1c04d6d7a50a030fc866f09d516020ef82324afae");
        byte[] g2x6b = hexToBytes("83f4b4e761936d90fd5f55f99087138a07a69755ad4a46e4dd1c2cfe6d11371e1cc033111a0595e3bba98d0f538db45119e384121b7d70927c49e6d044fd8517c36bc6ed2813a8956dd64f049869e8a77f7e46930240e6984abe26fa6a89658f");

        byte[] out = Context.ecAdd("bls12-381-g2", concatBytes(g2b, g2x2b, g2x3b), true);
        Context.require(Arrays.equals(g2x6b, out), "incorrect ecAdd.G2 result");
        
        Context.println("testBLS12381ecAddG2Compressed - OK");
    }

    public void testBLS12381ecScalarMulG2() {
        byte[] g2b = hexToBytes("024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb813e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e0ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b828010606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be");
        byte[] g2x2b = hexToBytes("1638533957d540a9d2370f17cc7ed5863bc0b995b8825e0ee1ea1e1e4d00dbae81f14b0bf3611b78c952aacab827a0530a4edef9c1ed7f729f520e47730a124fd70662a904ba1074728114d1031e1572c6c886f6b57ec72a6178288c47c335770468fb440d82b0630aeb8dca2b5256789a66da69bf91009cbfe6bd221e47aa8ae88dece9764bf3bd999d95d71e4c98990f6d4552fa65dd2638b361543f887136a43253d9c66c411697003f7a13c308f5422e1aa0a59c8967acdefd8b6e36ccf3");
        byte[] g2x6b = hexToBytes("19e384121b7d70927c49e6d044fd8517c36bc6ed2813a8956dd64f049869e8a77f7e46930240e6984abe26fa6a89658f03f4b4e761936d90fd5f55f99087138a07a69755ad4a46e4dd1c2cfe6d11371e1cc033111a0595e3bba98d0f538db45117a31a4fccfb5f768a2157517c77a4f8aaf0dee8f260d96e02e1175a8754d09600923beae02a019afc327b65a2fdbbfc088bb5832f4a4a452edda646ebaa2853a54205d56329960b44b2450070734724a74daaa401879bad142132316e9b3401");

        byte[] out;
        
        out = Context.ecScalarMul("bls12-381-g2", new BigInteger("2").toByteArray(), g2b, false);
        Context.require(Arrays.equals(g2x2b, out), "incorrect ecAdd result");
        
        out = Context.ecScalarMul("bls12-381-g2", new BigInteger("3").toByteArray(), g2x2b, false);
        Context.require(Arrays.equals(g2x6b, out), "incorrect ecAdd result");

        Context.println("testBLS12381ecScalarMulG2 - OK");
    }

    public void testBLS12381ecScalarMulG2Compressed() {
        byte[] g2b = hexToBytes("93e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8");
        byte[] g2x2b = hexToBytes("aa4edef9c1ed7f729f520e47730a124fd70662a904ba1074728114d1031e1572c6c886f6b57ec72a6178288c47c335771638533957d540a9d2370f17cc7ed5863bc0b995b8825e0ee1ea1e1e4d00dbae81f14b0bf3611b78c952aacab827a053");
        byte[] g2x6b = hexToBytes("83f4b4e761936d90fd5f55f99087138a07a69755ad4a46e4dd1c2cfe6d11371e1cc033111a0595e3bba98d0f538db45119e384121b7d70927c49e6d044fd8517c36bc6ed2813a8956dd64f049869e8a77f7e46930240e6984abe26fa6a89658f");

        byte[] out;
        
        out = Context.ecScalarMul("bls12-381-g2", new BigInteger("2").toByteArray(), g2b, true);
        Context.require(Arrays.equals(g2x2b, out), "incorrect ecScalarMul result");
        
        out = Context.ecScalarMul("bls12-381-g2", new BigInteger("3").toByteArray(), g2x2b, true);
        Context.require(Arrays.equals(g2x6b, out), "incorrect ecScalarMul result");

        Context.println("testBLS12381ecScalarMulG2Compressed - OK");
    }

    public void testBLS12381ecPairingCheck() {
        byte[] g1b = hexToBytes("17f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb08b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1");
        byte[] g1Negb = hexToBytes("17f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb114d1d6855d545a8aa7d76c8cf2e21f267816aef1db507c96655b9d5caac42364e6f38ba0ecb751bad54dcd6b939c2ca");
        byte[] g2b = hexToBytes("024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb813e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e0ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b828010606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be");
        byte[] g2Negb = hexToBytes("024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb813e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e0d1b3cc2c7027888be51d9ef691d77bcb679afda66c73f17f9ee3837a55024f78c71363275a75d75d86bab79f74782aa13fa4d4a0ad8b1ce186ed5061789213d993923066dddaf1040bc3ff59f825c78df74f2d75467e25e0f55f8a00fa030ed");
    
        boolean res = Context.ecPairingCheck("bls12-381", concatBytes(
            g1b, g2b,
            g1Negb, g2b,
            g1b, g2b,
            g1b, g2Negb
        ), false);
        Context.require(res, "incorrect ecPairingCheck result");

        Context.println("testBLS12381ecPairingCheck - OK");
    }

    public void testBLS12381ecPairingCheckCompressed() {
        byte[] g1b = hexToBytes("97f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb");
        byte[] g1Negb = hexToBytes("b7f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb");
        byte[] g2b = hexToBytes("93e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8");
        byte[] g2Negb = hexToBytes("b3e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8");
    
        boolean res = Context.ecPairingCheck("bls12-381", concatBytes(
            g1b, g2b,
            g1Negb, g2b,
            g1b, g2b,
            g1b, g2Negb
        ), true);
        Context.require(res, "incorrect ecPairingCheck result");

        Context.println("testBLS12381ecPairingCheckCompressed - OK");
    }

    public void testBLS12381InvalidDataEncoding() {
        byte[] g1b = hexToBytes("17f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb08b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e0");
        byte[] g2b = hexToBytes("024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb813e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e0ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b828010606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be");

        try {
            Context.ecAdd("bls12-381-g1", concatBytes(g1b, g1b), false);
            Context.require(false, "ecAddG1: should not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecAddG1)");
        }

        try {
            Context.ecAdd("bls12-381-g1", concatBytes(g1b, g1b), true);
            Context.require(false, "ecAddG1Compressed: shall not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecAddG1Compressed)");
        }

        try {
            Context.ecAdd("bls12-381-g2", concatBytes(g1b, g1b), false);
            Context.require(false, "ecAddG2: shall not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecAddG2)");
        }

        try {
            Context.ecAdd("bls12-381-g2", concatBytes(g1b, g1b), true);
            Context.require(false, "ecAddG2Compressed: shall not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecAddG2Compressed)");
        }

        try {
            Context.ecScalarMul("bls12-381-g1", new BigInteger("2").toByteArray(), concatBytes(g1b, g1b), false);
            Context.require(false, "ecScalarMulG1: shall not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecScalarMulG1)");
        }

        try {
            Context.ecScalarMul("bls12-381-g1", new BigInteger("2").toByteArray(), concatBytes(g1b, g1b), true);
            Context.require(false, "ecScalarMulG1Compressed: shall not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecScalarMulG1Compressed)");
        }

        try {
            Context.ecScalarMul("bls12-381-g2", new BigInteger("2").toByteArray(), concatBytes(g1b, g1b), false);
            Context.require(false, "ecScalarMulG2: shall not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecScalarMulG2)");
        }
        
        try {
            Context.ecScalarMul("bls12-381-g2", new BigInteger("2").toByteArray(), concatBytes(g1b, g1b), true);
            Context.require(false, "ecScalarMulG2Compressed: shall not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecScalarMulG2Compressed)");
        }

        try {
            Context.ecPairingCheck("bls12-381", concatBytes(g1b, g2b), false);
            Context.require(false, "ecPairingCheck: shall not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecPairingCheck)");
        }
        
        try {
            Context.ecPairingCheck("bls12-381", concatBytes(g1b, g2b), true);
            Context.require(false, "ecPairingCheckCompressed: shall not reach here");
        } catch (IllegalArgumentException e) {
            Context.println("testBLS12381InvalidPointEncoding - OK (ecPairingCheckCompressed)");
        }
    }


     private static byte[] concatBytes(byte[]... args) {
        int length = 0;
        for (int i = 0; i < args.length; i++) {
            length += args[i].length;
        }
        byte[] out = new byte[length];
        int offset = 0;
        for (int i = 0; i < args.length; i++) {
            System.arraycopy(args[i], 0, out, offset, args[i].length);
            offset += args[i].length;
        }
        return out;
    }
}
