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
import foundation.icon.ee.types.Method;

public class TxFrame implements Frame {
    private State state;
    private Account calleeAccount;
    private Contract calleeContract;

    public static TxFrame newCallFrame(State state, Address to) {
        var stateCopy = new State(state);
        return new TxFrame(stateCopy, to);
    }

    public static TxFrame newDeployFrame(State state,
            Address to,
            byte[] contractID,
            String codeID,
            Method[] methods,
            InvokeHandler ih) {
        var stateCopy = new State(state);
        stateCopy.deploy(to, new Contract(contractID, codeID, methods, ih));
        return new TxFrame(stateCopy, to);
    }

    private TxFrame(State state, Address address) {
        this.state = state;
        this.calleeAccount = state.getAccount(address);
        this.calleeContract = calleeAccount.getContract();
    }

    public State getState() {
        return state;
    }

    public void setState(State state) {
        this.state = state;
        this.calleeAccount = state.getAccount(getAddress());
        this.calleeContract = state.getContract(calleeContract.getID());
    }

    public Account getAccount() {
        return calleeAccount;
    }

    public Contract getContract() {
        return calleeContract;
    }

    public byte[] getContractID() {
        return calleeContract.getID();
    }

    public Address getAddress() {
        return calleeAccount.getAddress();
    }
}
