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

package foundation.icon.ee;

import foundation.icon.ee.test.Jars;
import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.test.TransactionException;
import foundation.icon.ee.types.Status;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.RevertedException;
import score.annotation.External;

import java.io.IOException;

public class DeployTest2 extends SimpleTest {
    public static class ValueHolder {
        private String value;

        public ValueHolder(String value) {
            this.value = value;
        }

        @External(readonly = true)
        public String getValue() {
            return value;
        }

        @External
        public void setValue(String value) {
            this.value = value;
        }
    }

    public static class Caller1 {
        public Caller1(Address valueHolder) {
            Context.call(valueHolder, "setValue", "in Caller1 constructor");
        }
    }

    public static class Caller2 {
        public Caller2(Address valueHolder) {
            Context.call(valueHolder, "setValue", "in Caller2 constructor");
            throw new IllegalArgumentException("Fail");
        }
    }

    public static class Caller3 {
        public Caller3(Address valueHolder) {
            Context.call(valueHolder, "setValue", "in Caller3 constructor");
        }
    }

    @Test
    public void external() {
        createAndAcceptNewJAVAEE();
        var valueHolder = sm.mustDeploy(ValueHolder.class, "initial");
        sm.setIndexer((addr) -> {
            if (addr.equals(valueHolder.getAddress())) {
                return 0;
            }
            return 1;
        });
        var caller1 = sm.mustDeploy(Caller1.class, valueHolder.getAddress());
        var res = valueHolder.query("getValue");
        Assertions.assertEquals("in Caller1 constructor", res.getRet());
        Assertions.assertThrows(TransactionException.class, ()-> {
            sm.mustDeploy(Caller2.class, valueHolder.getAddress());
        });
        res = valueHolder.query("getValue");
        Assertions.assertEquals("in Caller1 constructor", res.getRet());
    }

    public static class Deployer {
        @External
        public Address deploy(byte[] jar, Address param) {
            try {
                return Context.deploy(jar, param);
            } catch (RevertedException e) {
                e.printStackTrace();
                return null;
            }
        }

        @External
        public Address update(Address addr, byte[] jar, Address param) {
            try {
                return Context.deploy(addr, jar, param);
            } catch (RevertedException e) {
                e.printStackTrace();
                return null;
            }
        }
    }

    @Test
    public void internal() {
        createAndAcceptNewJAVAEE();
        var valueHolder = sm.mustDeploy(ValueHolder.class, "initial");
        var deployer = sm.mustDeploy(Deployer.class);
        sm.setIndexer((addr) -> {
            if (addr.equals(valueHolder.getAddress())
                    || addr.equals(deployer.getAddress())) {
                return 0;
            }
            return 1;
        });
        var caller1Code = Jars.make(Caller1.class);
        var res = deployer.invoke("deploy", caller1Code, valueHolder.getAddress());
        Assertions.assertNotNull(res.getRet());

        res = valueHolder.query("getValue");
        Assertions.assertEquals("in Caller1 constructor", res.getRet());

        var caller2Code = Jars.make(Caller2.class);
        res = deployer.invoke("deploy", caller2Code, valueHolder.getAddress());
        Assertions.assertNull(res.getRet());

        res = valueHolder.query("getValue");
        Assertions.assertEquals("in Caller1 constructor", res.getRet());
    }

    @Test
    public void internalUpdate() {
        createAndAcceptNewJAVAEE();
        var valueHolder = sm.mustDeploy(ValueHolder.class, "initial");
        var deployer = sm.mustDeploy(Deployer.class);
        sm.setIndexer((addr) -> {
            if (addr.equals(valueHolder.getAddress())
                    || addr.equals(deployer.getAddress())) {
                return 0;
            }
            return 1;
        });
        var caller1Code = Jars.make(Caller1.class);
        var res = deployer.invoke("deploy", caller1Code, valueHolder.getAddress());
        Assertions.assertNotNull(res.getRet());
        var childAddress = (foundation.icon.ee.types.Address) res.getRet();

        res = valueHolder.query("getValue");
        Assertions.assertEquals("in Caller1 constructor", res.getRet());

        var caller2Code = Jars.make(Caller2.class);
        res = deployer.invoke("update", childAddress, caller2Code, valueHolder.getAddress());
        Assertions.assertNull(res.getRet());

        res = valueHolder.query("getValue");
        Assertions.assertEquals("in Caller1 constructor", res.getRet());

        var caller3Code = Jars.make(Caller3.class);
        res = deployer.invoke("update", childAddress, caller3Code, valueHolder.getAddress());
        Assertions.assertNotNull(res.getRet());

        res = valueHolder.query("getValue");
        Assertions.assertEquals("in Caller3 constructor", res.getRet());
    }

    public static class Deployer2 {
        @External
        public String deployAndCall(byte[] jar, String ctorParam,
                String method) {
            Address child;
            try {
                child = Context.deploy(jar, ctorParam);
            } catch (RevertedException e) {
                e.printStackTrace();
                return null;
            }
            return Context.call(String.class, child, method);
        }

        @External
        public String updateAndCall(Address addr, byte[] jar, String ctorParam,
                String method) {
            Address child;
            try {
                child = Context.deploy(addr, jar, ctorParam);
            } catch (RevertedException e) {
                e.printStackTrace();
                return null;
            }
            return Context.call(String.class, child, method);
        }
    }

    @Test
    public void deployAndCall() {
        var deployer = sm.mustDeploy(Deployer2.class);
        var valueHolderCode = Jars.make(ValueHolder.class);
        var res = deployer.invoke("deployAndCall",
                valueHolderCode, "value", "getValue");
        Assertions.assertEquals("value", res.getRet());
    }

    @Test
    public void updateAndCall() {
        var deployer = sm.mustDeploy(Deployer2.class);
        // deploy any contract
        var dapp1 = sm.mustDeploy(Deployer2.class);
        var valueHolderCode = Jars.make(ValueHolder.class);
        var res = deployer.invoke("updateAndCall",
                dapp1.getAddress(), valueHolderCode, "value", "getValue");
        Assertions.assertEquals("value", res.getRet());
    }

    @Test
    void constantDynamic() throws IOException {
        var jar = readResourceFile("constant-dynamic.jar");
        var res = sm.tryDeploy(jar);
        Assertions.assertEquals(Status.IllegalFormat, res.getStatus());
    }
}
