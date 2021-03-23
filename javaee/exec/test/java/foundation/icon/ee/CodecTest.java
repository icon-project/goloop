package foundation.icon.ee;

import foundation.icon.ee.test.GoldenTest;
import org.junit.jupiter.api.Test;
import score.ByteArrayObjectWriter;
import score.Context;
import score.DictDB;
import score.ObjectReader;
import score.ObjectWriter;
import score.annotation.External;

public class CodecTest extends GoldenTest {
    public static class User {
        String name;
        int visitCount;
        String desc; // suppose previous version didn't have this field.

        public User(String name, int visitCount, String desc) {
            this.name = name;
            this.visitCount = visitCount;
            this.desc = desc;
        }

        public String toString() {
            return "User{" +
                    "name='" + name + '\'' +
                    ", visitCount=" + visitCount +
                    ", desc='" + desc + '\'' +
                    '}';
        }

        public static void writeObject(ObjectWriter w, User v) {
            w.writeListOf(
                    v.name,
                    v.visitCount,
                    v.desc
            );
        }

        public static User readObject(ObjectReader r) {
            r.beginList();
            var res = new User(
                    r.readString(),
                    r.readInt(),
                    r.readOrDefault(String.class, null)
            );
            r.end();
            return res;
        }
    }

    public static class Score {
        private DictDB<String, User> userDB = Context.newDictDB("userDB", User.class);

        @External
        public void run() {
            userDB.set("k1", new User("A", 10, "aaa"));
            User u = userDB.get("k1");
            Context.println(u.toString());
        }
    }

    @Test
    public void test() {
        var score = sm.mustDeploy(new Class<?>[]{Score.class, User.class});
        score.invoke("run");
    }
}
