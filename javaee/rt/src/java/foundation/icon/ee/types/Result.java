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

import i.AvmException;

import java.math.BigInteger;

public class Result {
    private final int status;
    private final BigInteger stepUsed;
    private final Object ret;

    public Result(int status, BigInteger stepUsed, Object ret) {
        this.status = status;
        this.stepUsed = stepUsed;
        this.ret = ret;
    }

    public Result(int status, long stepUsed, Object ret) {
        this.status = status;
        this.stepUsed = BigInteger.valueOf(stepUsed);
        this.ret = ret;
    }

    public int getStatus() {
        return status;
    }

    public BigInteger getStepUsed() {
        return stepUsed;
    }

    public Object getRet() {
        return ret;
    }

    @Override
    public String toString() {
        return "Result{" +
                "status=" + status +
                ", stepUsed=" + stepUsed +
                ", ret=" + ret +
                '}';
    }
}
