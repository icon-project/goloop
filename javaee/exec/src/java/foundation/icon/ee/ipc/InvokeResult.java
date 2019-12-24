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

package foundation.icon.ee.ipc;

import foundation.icon.ee.types.Result;

import java.io.IOException;
import java.math.BigInteger;

public class InvokeResult {

    private final int status;
    private final BigInteger stepUsed;
    private final TypedObj result;

    public InvokeResult(int status, BigInteger stepUsed, TypedObj result) {
        this.status = status;
        this.stepUsed = stepUsed;
        this.result = result;
    }

    public InvokeResult(Result result) throws IOException {
        this.status = result.getStatus();
        this.stepUsed = result.getStepUsed();
        this.result = TypedObj.encodeAny(result.getRet());
    }

    int getStatus() {
        return status;
    }

    BigInteger getStepUsed() {
        return stepUsed;
    }

    TypedObj getResult() {
        return result;
    }
}
