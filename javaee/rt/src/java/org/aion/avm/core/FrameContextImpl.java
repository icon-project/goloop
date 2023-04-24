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

package org.aion.avm.core;

import i.FrameContext;
import i.IDBStorage;

public class FrameContextImpl implements FrameContext {
    private final IExternalState externalState;
    private final IDBStorage dbs;
    private int flag;
    private boolean deployFrame;

    FrameContextImpl(IExternalState externalState) {
        this.externalState = externalState;
        this.dbs = new DBStorage(externalState);
    }

    FrameContextImpl(IExternalState externalState, boolean deploy) {
        this.externalState = externalState;
        this.dbs = new DBStorage(externalState);
        this.deployFrame = deploy;
    }

    public IDBStorage getDBStorage() {
        return dbs;
    }

    public IExternalState getExternalState() {
        return externalState;
    }

    public boolean waitForRefund() {
        return externalState.waitForCallback();
    }

    public void limitPendingRefundLength() {
        externalState.limitPendingCallbackLength();
    }

    public void setStatusFlag(int flag) {
        this.flag = flag;
    }

    public int getStatusFlag() {
        return flag;
    }

    public boolean isDeployFrame() {
        return deployFrame;
    }
}
