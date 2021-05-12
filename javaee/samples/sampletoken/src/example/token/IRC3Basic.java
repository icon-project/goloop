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

package example.token;

import example.util.EnumerableIntMap;
import example.util.IntSet;
import score.Address;
import score.Context;
import score.DictDB;
import score.annotation.EventLog;
import score.annotation.External;

import java.math.BigInteger;

public abstract class IRC3Basic implements IRC3 {
    protected static final Address ZERO_ADDRESS = new Address(new byte[Address.LENGTH]);
    private final String name;
    private final String symbol;
    private final DictDB<Address, IntSet> holderTokens = Context.newDictDB("holders", IntSet.class);
    private final EnumerableIntMap<Address> tokenOwners = new EnumerableIntMap<>("owners", Address.class);
    private final DictDB<BigInteger, Address> tokenApprovals = Context.newDictDB("approvals", Address.class);

    public IRC3Basic(String _name, String _symbol) {
        this.name = _name;
        this.symbol = _symbol;
    }

    @External(readonly=true)
    public String name() {
        return name;
    }

    @External(readonly=true)
    public String symbol() {
        return symbol;
    }

    @External(readonly=true)
    public int balanceOf(Address _owner) {
        Context.require(!ZERO_ADDRESS.equals(_owner));
        var tokens = holderTokens.get(_owner);
        return (tokens != null) ? tokens.length() : 0;
    }

    @External(readonly=true)
    public Address ownerOf(BigInteger _tokenId) {
        return tokenOwners.getOrThrow(_tokenId, "Non-existent token");
    }

    @External(readonly=true)
    public Address getApproved(BigInteger _tokenId) {
        return tokenApprovals.getOrDefault(_tokenId, ZERO_ADDRESS);
    }

    @External
    public void approve(Address _to, BigInteger _tokenId) {
        Address owner = ownerOf(_tokenId);
        Context.require(!owner.equals(_to));
        Context.require(owner.equals(Context.getCaller()));
        _approve(_to, _tokenId);
    }

    private void _approve(Address to, BigInteger tokenId) {
        tokenApprovals.set(tokenId, to);
        Approval(ownerOf(tokenId), to, tokenId);
    }

    @External
    public void transfer(Address _to, BigInteger _tokenId) {
        Address owner = ownerOf(_tokenId);
        Context.require(owner.equals(Context.getCaller()));
        _transfer(owner, _to, _tokenId);
    }

    @External
    public void transferFrom(Address _from, Address _to, BigInteger _tokenId) {
        Address owner = ownerOf(_tokenId);
        Address spender = Context.getCaller();
        Context.require(owner.equals(spender) || getApproved(_tokenId).equals(spender));
        _transfer(_from, _to, _tokenId);
    }

    private void _transfer(Address from, Address to, BigInteger tokenId) {
        Context.require(ownerOf(tokenId).equals(from));
        Context.require(!to.equals(ZERO_ADDRESS));
        // clear approvals from the previous owner
        _approve(ZERO_ADDRESS, tokenId);

        _removeTokenFrom(tokenId, from);
        _addTokenTo(tokenId, to);
        tokenOwners.set(tokenId, to);
        Transfer(from, to, tokenId);
    }

    /**
     * (Extension) Returns the total amount of tokens stored by the contract.
     */
    @External(readonly=true)
    public int totalSupply() {
        return tokenOwners.length();
    }

    /**
     * (Extension) Returns a token ID at a given index of all the tokens stored by the contract.
     * Use along with {@code _totalSupply} to enumerate all tokens.
     */
    @External(readonly=true)
    public BigInteger tokenByIndex(int _index) {
        return tokenOwners.getKey(_index);
    }

    /**
     * (Extension) Returns a token ID owned by owner at a given index of its token list.
     * Use along with {@code balanceOf} to enumerate all of owner's tokens.
     */
    @External(readonly=true)
    public BigInteger tokenOfOwnerByIndex(Address _owner, int _index) {
        var tokens = holderTokens.get(_owner);
        return (tokens != null) ? tokens.at(_index) : BigInteger.ZERO;
    }

    /**
     * Mints `tokenId` and transfers it to `to`.
     */
    protected void _mint(Address to, BigInteger tokenId) {
        Context.require(!ZERO_ADDRESS.equals(to));
        Context.require(!_tokenExists(tokenId));

        _addTokenTo(tokenId, to);
        tokenOwners.set(tokenId, to);
        Transfer(ZERO_ADDRESS, to, tokenId);
    }

    /**
     * Destroys `tokenId`.
     */
    protected void _burn(BigInteger tokenId) {
        Address owner = ownerOf(tokenId);
        // clear approvals
        _approve(ZERO_ADDRESS, tokenId);

        _removeTokenFrom(tokenId, owner);
        tokenOwners.remove(tokenId);
        Transfer(owner, ZERO_ADDRESS, tokenId);
    }

    private boolean _tokenExists(BigInteger tokenId) {
        return tokenOwners.contains(tokenId);
    }

    private void _addTokenTo(BigInteger tokenId, Address to) {
        var tokens = holderTokens.get(to);
        if (tokens == null) {
            tokens = new IntSet(to.toString());
            holderTokens.set(to, tokens);
        }
        tokens.add(tokenId);
    }

    private void _removeTokenFrom(BigInteger tokenId, Address from) {
        var tokens = holderTokens.get(from);
        Context.require(tokens != null);
        tokens.remove(tokenId);
        if (tokens.length() == 0) {
            holderTokens.set(from, null);
        }
    }

    @EventLog(indexed=3)
    public void Transfer(Address _from, Address _to, BigInteger _tokenId) {
    }

    @EventLog(indexed=3)
    public void Approval(Address _owner, Address _approved, BigInteger _tokenId) {
    }
}
