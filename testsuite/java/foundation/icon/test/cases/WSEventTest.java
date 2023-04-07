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

package foundation.icon.test.cases;

import foundation.icon.icx.IconService;
import foundation.icon.icx.KeyWallet;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Block;
import foundation.icon.icx.data.BlockNotification;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.EventNotification;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.http.HttpProvider;
import foundation.icon.icx.transport.monitor.EventMonitorSpec;
import foundation.icon.icx.transport.monitor.Monitor;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Env;
import foundation.icon.test.common.TransactionHandler;
import foundation.icon.test.score.EventGen;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.math.BigInteger;
import java.util.*;

import static foundation.icon.test.common.Env.LOG;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotNull;

// TODO What about adding annotation indicating requirements. For example,
// "@require(nodeNum=4,chainNum=1)" indicates it requires at least 4 nodes and
// 1 chain for each.
public class WSEventTest {
    private static TransactionHandler txHandler;
    private static IconService iconService;
    private final Object condVar = new Object();

    @BeforeAll
    public static void setUp() {
        Env.Node node = Env.nodes[0];
        Env.Channel channel = node.channels[0];
        Env.Chain chain = channel.chain;
        iconService = new IconService(new HttpProvider(channel.getAPIUrl(Env.testApiVer)));
        txHandler = new TransactionHandler(iconService, chain);
    }

    /*
    It receives 10 blocks from the last block.
    After that, it confirms whether the received blocks are transferred in order or whether the blocks are normal.
     */
    @Test
    public void wsBlkMonitorTest() throws Exception {
        Block lastBlk = iconService.getLastBlock().execute();
        BigInteger reqBlkHeight = lastBlk.getHeight().add(BigInteger.TEN);
        Monitor<BlockNotification> bm = iconService.monitorBlocks(reqBlkHeight);
        List<BlockNotification> notiList = new LinkedList<>();
        final int testBlkNum = 10;
        bm.start(new Monitor.Listener<BlockNotification>() {
            boolean complete = false;
            boolean stop = false;

            @Override
            public void onStart() {
                synchronized (condVar) {
                    assertFalse(stop);
                }
            }

            @Override
            public void onEvent(BlockNotification event) {
                synchronized (condVar) {
                    assertFalse(stop);
                    if(!complete) {
                        LOG.info("received block " + event.getHeight());
                        notiList.add(event);
                        if(notiList.size() == testBlkNum) {
                            complete = true;
                            condVar.notify();
                        }
                    }
                }
            }

            @Override
            public void onError(long code) {
                throw new RuntimeException("onError code : " + code);
            }

            @Override
            public void onClose() {
                synchronized (condVar) {
                    assertFalse(stop);
                    stop = true;
                    notiList.clear();
                    condVar.notify();
                }
            }
        });

        BlockNotification noti = null;
        synchronized (condVar) {
            condVar.wait(3000 * testBlkNum);
            assertEquals(testBlkNum, notiList.size());
        }
        for(int i = 0; i < testBlkNum; i++) {
            noti = notiList.get(i);
            assertNotNull(noti);
            LOG.infoEntering("check received block " + noti.getHeight());
            // check the order og the received blocks
            int cmp = noti.getHeight().compareTo(reqBlkHeight.add(BigInteger.valueOf(i)));
            assertEquals(0, cmp);
            Block blk = iconService.getBlock(noti.getHash()).execute();
            cmp = blk.getHeight().compareTo(noti.getHeight());
            assertEquals(0, cmp);
            LOG.infoExiting();
        }

        LOG.infoEntering("stop");
        bm.stop();
        synchronized (condVar) {
            condVar.wait(3000);
            assertEquals(0, notiList.size());
        }
        LOG.infoExiting();
    }

    @Test
    @Tag(Constants.TAG_PY_SCORE)
    public void wsBlkMonitorWithEventFiltersTestWithPython() throws Exception {
        wsBlkMonitorWithEventFiltersTest(Constants.CONTENT_TYPE_PYTHON);
    }

    @Test
    @Tag(Constants.TAG_JAVA_SCORE)
    public void wsBlkMonitorWithEventFiltersTestWithJava() throws Exception {
        wsBlkMonitorWithEventFiltersTest(Constants.CONTENT_TYPE_JAVA);
    }

    public void wsBlkMonitorWithEventFiltersTest(String contentType) throws Exception {
        KeyWallet ownerWallet = KeyWallet.create();
        KeyWallet aliceWallet = KeyWallet.create();
        KeyWallet bobWallet = KeyWallet.create();

        // deploy 2 scores with same source
        EventGen[] eventGen = new EventGen[2];
        LOG.infoEntering("deploy", "event gen SCORE");
        eventGen[0] = EventGen.install(txHandler, ownerWallet, contentType);
        LOG.infoExiting();

        LOG.infoEntering("deploy", "event gen SCORE");
        eventGen[1] = EventGen.install(txHandler, ownerWallet, contentType);
        LOG.infoExiting();

        String event = "Event(Address,int,bytes)";
        Address[] addrs = new Address[] {
                null, eventGen[0].getAddress(), eventGen[0].getAddress(),
        };
        String[][] data = new String[][] {
                null,
                null,
                {bobWallet.getAddress().toString(), "500", "0x0A"},
        };
        LOG.info("bobAddr : " + bobWallet.getAddress().toString());
        int[] expectedEventNum = {4,2,1};

        for(int i = 0; i < 3; i++) {
            Map<BigInteger, BigInteger[][]> eventIndexesMap = new HashMap<>();
            LOG.infoEntering("request monitor[" + i + "]");
            Block lastBlk = iconService.getLastBlock().execute();

            /*
            1. monitor with event
            2. monitor with event and address
            3. monitor with event, address and data
             */
            EventMonitorSpec.EventFilter eventFilter = new EventMonitorSpec.EventFilter(
                    event, addrs[i], data[i], null);
            Monitor<BlockNotification> em = iconService.monitorBlocks(lastBlk.getHeight(), new EventMonitorSpec.EventFilter[]{eventFilter});
            boolean started = em.start(new Monitor.Listener<BlockNotification>() {
                boolean stop = false;

                @Override
                public void onStart() {
                    synchronized (condVar) {
                        assertFalse(stop);
                    }
                }

                @Override
                public void onEvent(BlockNotification event) {
                    synchronized (condVar) {
                        assertFalse(stop);
                        LOG.info("receive height : " + event.getHeight() + ", numOfTx : " +
                                (event.getIndexes() != null ? event.getIndexes()[0].length : 0));
                        if (event.getIndexes() != null) {
                            eventIndexesMap.put(event.getHeight(), event.getEvents()[0]);
                        }
                    }
                }

                @Override
                public void onError(long code) {
                    LOG.warning("onError code : " + code);
                    throw new RuntimeException();
                }

                @Override
                public void onClose() {
                    synchronized (condVar) {
                        assertFalse(stop);
                        stop = true;
                    }
                }
            });
            if(!started) {
                throw new IllegalStateException();
            }

            for(EventGen eg : eventGen) {
                Bytes txHash = eg.invokeGenerate(aliceWallet, bobWallet.getAddress(), new BigInteger("100"), new byte[]{1});
                LOG.info("sendTx : " + txHash);
            }

            for(EventGen eg : eventGen) {
                TransactionResult txResult = eg.invokeGenerateAndWait(aliceWallet, bobWallet.getAddress(), new BigInteger(data[2][1]), new byte[]{Byte.decode(data[2][2])});
                LOG.info("sendTx : " + txResult.getTxHash());
            }

            synchronized (condVar) {
                condVar.wait(5000);
                Optional<Integer> numOfEvent = eventIndexesMap.values().stream()
                        .map((l) -> Arrays.stream(l)
                                .map((e) -> e.length)
                                .reduce(Integer::sum)
                                .orElse(0))
                        .reduce(Integer::sum);
                assertEquals(expectedEventNum[i], numOfEvent.orElse(0));
            }
            em.stop();
            LOG.infoExiting("close");

            try {
                em.stop();
            }
            catch(IllegalStateException ex) {
                LOG.info(ex.getMessage());
            }
        }
    }

    @Test
    @Tag(Constants.TAG_PY_SCORE)
    public void wsEvtMonitorTestWithPython() throws Exception {
        wsEvtMonitorTest(Constants.CONTENT_TYPE_PYTHON);
    }

    @Test
    @Tag(Constants.TAG_JAVA_SCORE)
    public void wsEvtMonitorTestWithJava() throws Exception {
        wsEvtMonitorTest(Constants.CONTENT_TYPE_JAVA);
    }

    public void wsEvtMonitorTest(String contentType) throws Exception {
        KeyWallet ownerWallet = KeyWallet.create();
        KeyWallet aliceWallet = KeyWallet.create();
        KeyWallet bobWallet = KeyWallet.create();
        final long PROGRESS_INTERVAL = 3;

        // deploy 2 scores with same source
        EventGen[] eventGen = new EventGen[2];
        LOG.infoEntering("deploy", "event gen SCORE");
        eventGen[0] = EventGen.install(txHandler, ownerWallet, contentType);
        LOG.infoExiting();

        LOG.infoEntering("deploy", "event gen SCORE");
        eventGen[1] = EventGen.install(txHandler, ownerWallet, contentType);
        LOG.infoExiting();

        String event = "Event(Address,int,bytes)";
        Address[] addrs = new Address[] {
                null, eventGen[0].getAddress(), eventGen[0].getAddress(),
        };
        String[][] data = new String[][] {
                null,
                null,
                {bobWallet.getAddress().toString(), "500", "0x0A"},
        };
        LOG.info("bobAddr : " + bobWallet.getAddress().toString());
        int[] expectedEventNum = {4,2,1};

        for(int i = 0; i < 3; i++) {
            List<EventNotification> eventList = new LinkedList<>();
            LOG.infoEntering("request monitor[" + i + "]");
            Block lastBlk = iconService.getLastBlock().execute();
            /*
            1. monitor with event
            2. monitor with event and address
            3. monitor with event, address and data
             */
            Monitor<EventNotification> em = iconService.monitor(new EventMonitorSpec(
                    lastBlk.getHeight(), event, addrs[i], data[i], null, false, PROGRESS_INTERVAL
            ));
            boolean started = em.start(new Monitor.Listener<EventNotification>() {
                boolean stop = false;
                BigInteger last = null;

                @Override
                public void onStart() {
                    synchronized (condVar) {
                        assertFalse(stop);
                    }
                }

                @Override
                public void onEvent(EventNotification event) {
                    synchronized (condVar) {
                        assertFalse(stop);
                        LOG.info("receive height : " + event.getHeight() + ", index : " + event.getIndex());
                        eventList.add(event);
                    }
                }

                @Override
                public void onProgress(BigInteger height) {
                    synchronized (condVar) {
                        assertFalse(stop);
                        if (last != null) {
                            var distance = height.subtract(last).longValue();
                            assertFalse(distance > PROGRESS_INTERVAL);
                        }
                        last = height;
                        LOG.info("receive progress height : " + height.toString());
                    }
                }

                @Override
                public void onError(long code) {
                    throw new RuntimeException();
                }

                @Override
                public void onClose() {
                    synchronized (condVar) {
                        assertFalse(stop);
                        stop = true;
                    }
                }
            });
            if(!started) {
                throw new IllegalStateException();
            }

            for(EventGen eg : eventGen) {
                Bytes txHash = eg.invokeGenerate(aliceWallet, bobWallet.getAddress(), new BigInteger("100"), new byte[]{1});
                LOG.info("sendTx : " + txHash);
            }

            for(EventGen eg : eventGen) {
                TransactionResult txResult = eg.invokeGenerateAndWait(aliceWallet, bobWallet.getAddress(), new BigInteger(data[2][1]), new byte[]{Byte.decode(data[2][2])});
                LOG.info("sendTx : " + txResult.getTxHash());
            }

            synchronized (condVar) {
                condVar.wait(5000);
                assertEquals(expectedEventNum[i], eventList.size());
            }
            em.stop();
            LOG.infoExiting("close");

            try {
                em.stop();
            }
            catch(IllegalStateException ex) {
                LOG.info(ex.getMessage());
            }
        }
    }
}
