package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.Address;
import score.BranchDB;
import score.Context;
import score.DictDB;
import score.ObjectReader;
import score.ObjectWriter;
import score.annotation.External;

import java.math.BigInteger;

public class KVDBTest extends GoldenTest {
    public static class Score {
        public void runVarDBExample() {
            var counter = Context.newVarDB("counter", BigInteger.class);
            counter.set(BigInteger.ZERO);
            counter.set(counter.get().add(BigInteger.ONE));
        }

        public void runArrayDBExample() {
            var addrList = Context.newArrayDB("addrList", Address.class);
            addrList.add(Context.getAddress());
            Context.require(addrList.size() == 1);
            Context.require(addrList.get(0).equals(Context.getAddress()));
            addrList.set(0, Context.getOwner());
            Context.require(addrList.get(0).equals(Context.getOwner()));
        }

        public void runDictDBExample() {
            DictDB<Address, BigInteger> balances;
            balances = Context.newDictDB("balances", BigInteger.class);
            var balance = BigInteger.valueOf(1_000_000);
            balances.set(Context.getOwner(), balance);
            Context.require(balances.get(Context.getOwner()).equals(balance));
        }

        public void runBranchDBExample() {
            BranchDB<BigInteger, DictDB<Address, Boolean>> confirmations;
            confirmations = Context.newBranchDB("confirmations", Boolean.class);

            var txID = BigInteger.ZERO;
            confirmations.at(txID).set(Context.getCaller(), true);
            Context.require(confirmations.at(txID).get(Context.getCaller()));
        }

        public static class Transaction {
            private final Address from;
            private final Address to;
            private final BigInteger value;

            public Transaction(Address from, Address to, BigInteger value) {
                this.from = from;
                this.to = to;
                this.value = value;
            }

            public Address getFrom() {
                return from;
            }

            public Address getTo() {
                return to;
            }

            public BigInteger getValue() {
                return value;
            }

            public static void writeObject(ObjectWriter w, Transaction t) {
                w.writeListOf(t.from, t.to, t.value);
            }

            public static Transaction readObject(ObjectReader r) {
                r.beginList();
                var t = new Transaction(
                        r.read(Address.class),
                        r.read(Address.class),
                        r.read(BigInteger.class)
                );
                r.end();
                return t;
            }

            @Override
            public boolean equals(Object o) {
                if (this == o) return true;
                if (o == null || getClass() != o.getClass()) return false;
                Transaction that = (Transaction) o;
                return from.equals(that.from) &&
                        to.equals(that.to) &&
                        value.equals(that.value);
            }
        }

        public void runUserClassExample() {
            DictDB<BigInteger, Transaction> transactions;
            transactions = Context.newDictDB("transactions", Transaction.class);

            var tx1 = new Transaction(
                    Context.getOrigin(),
                    Context.getOwner(),
                    BigInteger.valueOf(1_000)
            );
            transactions.set(BigInteger.valueOf(1), tx1);

            var tx2 = transactions.get(BigInteger.valueOf(1));
            Context.require(tx1.equals(tx2));
        }

        @External
        public void run() {
            runVarDBExample();
            runArrayDBExample();
            runDictDBExample();
            runBranchDBExample();
            runUserClassExample();
        }
    }

    @Test
    void testExampleScore() {
        var score = sm.mustDeploy(Score.class);
        score.invoke("run");
    }
}
