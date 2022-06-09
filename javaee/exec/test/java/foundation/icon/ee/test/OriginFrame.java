/*
 * Copyright 2020 ICON Foundation
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

package foundation.icon.ee.test;

import foundation.icon.ee.types.Address;

public class OriginFrame implements Frame {
    private State state;
    private Address address;

    public OriginFrame(Address address) {
        this.address = address;
        this.state = new State();
    }

    public OriginFrame(Address address, State state) {
        this.address = address;
        this.state = state;
    }

    @Override
    public State getState() {
        return state;
    }

    @Override
    public void setState(State state) {
        this.state = state;
    }

    @Override
    public Account getAccount() {
        throw new UnsupportedOperationException();
    }

    @Override
    public Contract getContract() {
        throw new UnsupportedOperationException();
    }

    @Override
    public byte[] getContractID() {
        throw new UnsupportedOperationException();
    }

    @Override
    public Address getAddress() {
        return address;
    }

    public void setAddress(Address address) {
        this.address = address;
    }
}
