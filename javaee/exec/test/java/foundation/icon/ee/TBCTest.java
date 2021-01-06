package foundation.icon.ee;

import foundation.icon.ee.test.ContractAddress;
import foundation.icon.ee.test.SimpleTest;
import foundation.icon.ee.test.TBCProtocol;
import foundation.icon.ee.test.TBCTestScenario;
import foundation.icon.ee.test.TBCTestScenarios;
import org.junit.jupiter.api.Test;
import test.TBCInterpreter;

import java.math.BigInteger;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class TBCTest extends SimpleTest {
    private static final int S = TBCProtocol.VAR_TYPE_STATIC;
    private static final int I = TBCProtocol.VAR_TYPE_INSTANCE;
    private static final int L = TBCProtocol.VAR_TYPE_LOCAL;

    void subcase(String name, ContractAddress c, TBCTestScenario scenario) {
        var totalExp = scenario.getExpectCount();
        var tr = c.invoke("run", (Object)scenario.compile());
        var okObs = (BigInteger)tr.getRet();
        assertEquals(BigInteger.valueOf(totalExp), okObs);
    }

    @Test
    public void test() {
        createAndAcceptNewJAVAEE();
        var c1 = sm.mustDeploy(TBCInterpreter.class, "c1");
        sm.setIndexer((addr) -> 1);
        var c2 = sm.mustDeploy(TBCInterpreter.class, "c2");
        sm.setIndexer((addr) -> {
            if (addr.equals(c1.getAddress())) {
                return 0;
            }
            return 1;
        });
        var a1 = c1.getAddress().toByteArray();
        var a2 = c2.getAddress().toByteArray();
        subcase("Static value direct", c1,
                TBCTestScenarios.newValueScenario(S, a1, a1));
        subcase("Instance value direct", c1, TBCTestScenarios.newValueScenario(I, a1, a1));
        subcase("Static value indirect", c1, TBCTestScenarios.newValueScenario(S, a1, a2));
        subcase("Instance value indirect", c1, TBCTestScenarios.newValueScenario(I, a1, a2));

        subcase("Static ref direct", c1, TBCTestScenarios.newRefScenario(S, a1, a1));
        subcase("Instance ref direct", c1, TBCTestScenarios.newRefScenario(I, a1, a1));
        subcase("Static ref indirect", c1, TBCTestScenarios.newRefScenario(S, a1, a2));
        subcase("Instance ref indirect", c1, TBCTestScenarios.newRefScenario(I, a1, a2));

        subcase("Local/Instance ref direct", c1, TBCTestScenarios.newLocalRefScenario(I, a1, a1));
        subcase("Local/Instance ref indirect", c1, TBCTestScenarios.newLocalRefScenario(I, a1, a2));
        subcase("Local/Static ref direct", c1, TBCTestScenarios.newLocalRefScenario(I, a1, a1));
        subcase("Local/Static ref indirect", c1, TBCTestScenarios.newLocalRefScenario(I, a1, a2));
    }
}
