package foundation.icon.ee;

import foundation.icon.ee.test.Jars;
import foundation.icon.ee.test.SimpleTest;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;
import score.Address;
import score.Context;
import score.annotation.External;

public class UpdateTest extends SimpleTest {
    public static class CodeSelf {
        private final String name;

        public CodeSelf(String name) {
            this.name = name;
        }

        @External(readonly = true)
        public String getName() {
            return name;
        }

        @External
        public String update(byte[] jar, String name) {
            Context.deploy(Context.getAddress(), jar, name);
            return this.name;
        }
    }

    public static class Parent {
        private Address child;

        @External
        public void updateChild(byte[] jar) {
            child = Context.deploy(child, jar);
        }

        @External
        public String callChild() {
            return Context.call(String.class, child, "getCodeName");
        }
    }

    public static class CodeA {
        @External(readonly = true)
        public String getCodeName() {
            return "CodeA";
        }
    }

    public static class CodeB {
        @External(readonly = true)
        public String getCodeName() {
            return "CodeB";
        }
    }

    @Test
    public void selfSame() {
        var jar = Jars.make(CodeSelf.class);
        var c = sm.mustDeploy(jar, "name1");
        var res = c.invoke("update", jar, "name2");
        Assertions.assertEquals("name1", res.getRet());
        res = c.invoke("getName");
        Assertions.assertEquals("name2", res.getRet());
    }

    @Test
    public void ab() {
        var parent = sm.mustDeploy(Parent.class);
        var codeA = Jars.make(CodeA.class);
        parent.invoke("updateChild", (Object) codeA);
        var res = parent.invoke("callChild");
        Assertions.assertEquals("CodeA", res.getRet());
        var codeB = Jars.make(CodeB.class);
        parent.invoke("updateChild", (Object) codeB);
        res = parent.invoke("callChild");
        Assertions.assertEquals("CodeB", res.getRet());
    }
}
