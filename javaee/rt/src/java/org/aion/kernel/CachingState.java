package org.aion.kernel;

import foundation.icon.ee.types.Address;
import foundation.icon.ee.types.Result;
import i.RuntimeAssertionError;
import org.aion.avm.core.IExternalState;
import org.aion.data.IAccountStore;
import org.aion.data.IDataStore;
import org.aion.data.MemoryBackedDataStore;

import java.math.BigInteger;

/**
 * In in-memory cached used by the TransactionalState in order to store results of in-flight transactions prior to commit.
 */
public class CachingState implements IExternalState {
    private final IDataStore dataStore;

    /**
     * Creates an instance which is backed by in-memory structures, only.
     */
    public CachingState() {
        this.dataStore = new MemoryBackedDataStore();
    }

    @Override
    public IExternalState newChildExternalState() {
        // While this kind of kernel could support children, the use-case would be an error, based on what this implementation is for.
        throw RuntimeAssertionError.unreachable("Caching state should never be asked to create children.");
    }

    @Override
    public void commit() {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public void commitTo(IExternalState target) {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public byte[] getBlockHashByHeight(long blockHeight) {
        throw RuntimeAssertionError.unreachable("No equivalent concept in the Avm.");
    }

    @Override
    public void removeStorage(Address address, byte[] key) {
        lazyCreateAccount(address.toByteArray()).removeData(key);
    }

    @Override
    public void createAccount(Address address) {
        this.dataStore.createAccount(address.toByteArray());
    }

    @Override
    public boolean hasAccountState(Address address) {
        return this.dataStore.openAccount(address.toByteArray()) != null;
    }

    @Override
    public byte[] getCode(Address address) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
            ? account.getCode()
            : null;
    }

    @Override
    public void putCode(Address address, byte[] code) {
        // Note that saving empty code is invalid since a valid JAR is not empty.
        RuntimeAssertionError.assertTrue((null != code) && (code.length > 0));
        lazyCreateAccount(address.toByteArray()).setCode(code);
    }

    @Override
    public byte[] getTransformedCode(Address address) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
            ? account.getTransformedCode()
            : null;
    }

    @Override
    public void setTransformedCode(Address address, byte[] code) {
        RuntimeAssertionError.assertTrue((null != code) && (code.length > 0));
        lazyCreateAccount(address.toByteArray()).setTransformedCode(code);
    }

    @Override
    public void putObjectGraph(Address address, byte[] bytes) {
        lazyCreateAccount(address.toByteArray()).setObjectGraph(bytes);
    }

    @Override
    public byte[] getObjectGraph(Address address) {
        return lazyCreateAccount(address.toByteArray()).getObjectGraph();
    }

    @Override
    public void putStorage(Address address, byte[] key, byte[] value) {
        lazyCreateAccount(address.toByteArray()).setData(key, value);
    }

    @Override
    public byte[] getStorage(Address address, byte[] key) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
                ? account.getData(key)
                : null;
    }

    @Override
    public void deleteAccount(Address address) {
        this.dataStore.deleteAccount(address.toByteArray());
    }

    @Override
    public BigInteger getBalance(Address address) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
                ? account.getBalance()
                : BigInteger.ZERO;
    }

    @Override
    public void adjustBalance(Address address, BigInteger delta) {
        internalAdjustBalance(address, delta);
    }

    @Override
    public BigInteger getNonce(Address address) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
                ? BigInteger.valueOf(account.getNonce())
                : BigInteger.ZERO;
    }

    @Override
    public void incrementNonce(Address address) {
        IAccountStore account = lazyCreateAccount(address.toByteArray());
        long start = account.getNonce();
        account.setNonce(start + 1);
    }

    @Override
    public boolean accountNonceEquals(Address address, BigInteger nonce) {
        return nonce.compareTo(this.getNonce(address)) == 0;
    }

    @Override
    public boolean accountBalanceIsAtLeast(Address address, BigInteger amount) {
        return this.getBalance(address).compareTo(amount) >= 0;
    }

    @Override
    public boolean isValidEnergyLimitForCreate(long energyLimit) {
        return energyLimit > 0;
    }

    @Override
    public boolean isValidEnergyLimitForNonCreate(long energyLimit) {
        return energyLimit > 0;
    }

    @Override
    public boolean destinationAddressIsSafeForThisVM(Address address) {
        // This implementation knows nothing of other VMs so it could only ever return true.
        // Since that is somewhat misleading (it assumes it is making a decision based on something), it is more reliable to just never call it.
        throw RuntimeAssertionError.unreachable("Caching kernel knows nothing of other VMs.");
    }

    @Override
    public long getBlockHeight() {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public long getBlockTimestamp() {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public Address getOwner() {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public void refundAccount(Address address, BigInteger amount) {
        // This method may have special logic in the kernel. Here it is just adjustBalance.
        internalAdjustBalance(address, amount);
    }

    private IAccountStore lazyCreateAccount(byte[] address) {
        IAccountStore account = this.dataStore.openAccount(address);
        if (null == account) {
            account = this.dataStore.createAccount(address);
        }
        return account;
    }

    private void internalAdjustBalance(Address address, BigInteger delta) {
        IAccountStore account = lazyCreateAccount(address.toByteArray());
        BigInteger start = account.getBalance();
        account.setBalance(start.add(delta));
    }

    @Override
    public void log(byte[][] indexed, byte[][] data) {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public Result call(Address address,
                       String method,
                       Object[] params,
                       BigInteger value,
                       int stepLimit) {
        return null;
    }
}
