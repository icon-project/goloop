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

package foundation.icon.ee.types;

import java.math.BigInteger;

public class Result {
    private final int status;
    private final BigInteger stepUsed;
    private final Object ret;
    private final int eid;
    private final int prevEID;

    public Result(int status, BigInteger stepUsed, Object ret, int eid,
            int prevEID) {
        this.status = status;
        this.stepUsed = stepUsed;
        this.ret = ret;
        this.eid = eid;
        this.prevEID = prevEID;
    }

    public Result(int status, BigInteger stepUsed, Object ret) {
        this(status, stepUsed, ret, 0, 0);
    }

    public Result(int status, long stepUsed, Object ret) {
        this(status, BigInteger.valueOf(stepUsed), ret, 0, 0);
    }

    public Result updateStatus(int status) {
        return new Result(status, stepUsed, ret, eid, prevEID);
    }

    public Result updateRet(Object ret) {
        return new Result(status, stepUsed, ret, eid, prevEID);
    }

    public int getStatus() {
        return status;
    }

    public int getStatusCode() {
        return status & Status.CodeMask;
    }

    public BigInteger getStepUsed() {
        return stepUsed;
    }

    public Object getRet() {
        return ret;
    }

    public int getEID() {
        return eid;
    }

    public int getPrevEID() {
        return prevEID;
    }

    @Override
    public String toString() {
        return "Result{" +
                "status=" + status +
                ", stepUsed=" + stepUsed +
                ", ret=" + ret +
                ", eid=" + eid +
                ", prevEID=" + prevEID +
                '}';
    }
}
