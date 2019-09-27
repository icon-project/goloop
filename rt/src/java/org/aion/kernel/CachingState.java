package org.aion.kernel;

import java.math.BigInteger;
import i.RuntimeAssertionError;
import org.aion.avm.core.IExternalState;
import org.aion.types.AionAddress;
import org.aion.data.IAccountStore;
import org.aion.data.IDataStore;
import org.aion.data.MemoryBackedDataStore;

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
    public byte[] getBlockHashByNumber(long blockNumber) {
        throw RuntimeAssertionError.unreachable("No equivalent concept in the Avm.");
    }

    @Override
    public void removeStorage(AionAddress address, byte[] key) {
        lazyCreateAccount(address.toByteArray()).removeData(key);
    }

    @Override
    public void createAccount(AionAddress address) {
        this.dataStore.createAccount(address.toByteArray());
    }

    @Override
    public boolean hasAccountState(AionAddress address) {
        return this.dataStore.openAccount(address.toByteArray()) != null;
    }

    @Override
    public byte[] getCode(AionAddress address) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
            ? account.getCode()
            : null;
    }

    @Override
    public void putCode(AionAddress address, byte[] code) {
        // Note that saving empty code is invalid since a valid JAR is not empty.
        RuntimeAssertionError.assertTrue((null != code) && (code.length > 0));
        lazyCreateAccount(address.toByteArray()).setCode(code);
    }

    @Override
    public byte[] getTransformedCode(AionAddress address) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
            ? account.getTransformedCode()
            : null;
    }

    @Override
    public void setTransformedCode(AionAddress address, byte[] code) {
        RuntimeAssertionError.assertTrue((null != code) && (code.length > 0));
        lazyCreateAccount(address.toByteArray()).setTransformedCode(code);
    }

    @Override
    public void putObjectGraph(AionAddress address, byte[] bytes) {
        lazyCreateAccount(address.toByteArray()).setObjectGraph(bytes);
    }

    @Override
    public byte[] getObjectGraph(AionAddress address) {
        return lazyCreateAccount(address.toByteArray()).getObjectGraph();
    }

    @Override
    public void putStorage(AionAddress address, byte[] key, byte[] value) {
        lazyCreateAccount(address.toByteArray()).setData(key, value);
    }

    @Override
    public byte[] getStorage(AionAddress address, byte[] key) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
                ? account.getData(key)
                : null;
    }

    @Override
    public void deleteAccount(AionAddress address) {
        this.dataStore.deleteAccount(address.toByteArray());
    }

    @Override
    public BigInteger getBalance(AionAddress address) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
                ? account.getBalance()
                : BigInteger.ZERO;
    }

    @Override
    public void adjustBalance(AionAddress address, BigInteger delta) {
        internalAdjustBalance(address, delta);
    }

    @Override
    public BigInteger getNonce(AionAddress address) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
                ? BigInteger.valueOf(account.getNonce())
                : BigInteger.ZERO;
    }

    @Override
    public void incrementNonce(AionAddress address) {
        IAccountStore account = lazyCreateAccount(address.toByteArray());
        long start = account.getNonce();
        account.setNonce(start + 1);
    }

    @Override
    public boolean accountNonceEquals(AionAddress address, BigInteger nonce) {
        return nonce.compareTo(this.getNonce(address)) == 0;
    }

    @Override
    public boolean accountBalanceIsAtLeast(AionAddress address, BigInteger amount) {
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
    public boolean destinationAddressIsSafeForThisVM(AionAddress address) {
        // This implementation knows nothing of other VMs so it could only ever return true.
        // Since that is somewhat misleading (it assumes it is making a decision based on something), it is more reliable to just never call it.
        throw RuntimeAssertionError.unreachable("Caching kernel knows nothing of other VMs.");
    }

    @Override
    public long getBlockNumber() {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public long getBlockTimestamp() {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public long getBlockEnergyLimit() {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public BigInteger getBlockDifficulty() {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public AionAddress getMinerAddress() {
        throw RuntimeAssertionError.unreachable("This class does not implement this method.");
    }

    @Override
    public void refundAccount(AionAddress address, BigInteger amount) {
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

    private void internalAdjustBalance(AionAddress address, BigInteger delta) {
        IAccountStore account = lazyCreateAccount(address.toByteArray());
        BigInteger start = account.getBalance();
        account.setBalance(start.add(delta));
    }
}
