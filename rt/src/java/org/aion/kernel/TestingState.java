package org.aion.kernel;

import java.math.BigInteger;
import org.aion.avm.core.IExternalState;
import org.aion.types.AionAddress;
import org.aion.avm.core.util.Helpers;
import org.aion.data.DirectoryBackedDataStore;
import org.aion.data.IAccountStore;
import org.aion.data.IDataStore;
import org.aion.data.MemoryBackedDataStore;

import java.io.File;


/**
 * A modified version of CachingState to support more general usage so it can be used as the external state underlying tests.
 */
public class TestingState implements IExternalState {
    /**
     * For testing purposes, we will give every contract address this prefix.
     */
    public static final byte AVM_CONTRACT_PREFIX = 0x0b;

    public static final AionAddress PREMINED_ADDRESS = new AionAddress(Helpers.hexStringToBytes("a025f4fd54064e869f158c1b4eb0ed34820f67e60ee80a53b469f725efc06378"));
    public static final AionAddress BIG_PREMINED_ADDRESS = new AionAddress(Helpers.hexStringToBytes("a035f4fd54064e869f158c1b4eb0ed34820f67e60ee80a53b469f725efc06378"));
    public static final BigInteger PREMINED_AMOUNT = BigInteger.TEN.pow(20);
    public static final BigInteger PREMINED_BIG_AMOUNT = BigInteger.valueOf(465000000).multiply(PREMINED_AMOUNT);
    public static long blockTimeMillis = 10_000L;

    private BigInteger blockDifficulty;
    private long blockNumber;
    private long blockTimestamp;
    private long blockNrgLimit;
    private AionAddress blockCoinbase;

    private final IDataStore dataStore;

    /**
     * Creates an instance of the interface which is backed by in-memory structures, only.
     */
    public TestingState() {
        this.dataStore = new MemoryBackedDataStore();
        IAccountStore premined = this.dataStore.createAccount(PREMINED_ADDRESS.toByteArray());
        premined.setBalance(PREMINED_AMOUNT);
        premined = this.dataStore.createAccount(BIG_PREMINED_ADDRESS.toByteArray());
        premined.setBalance(PREMINED_BIG_AMOUNT);
        this.blockDifficulty = BigInteger.valueOf(10_000_000L);
        this.blockNumber = 1;
        this.blockTimestamp = System.currentTimeMillis();
        this.blockNrgLimit = 10_000_000L;
        this.blockCoinbase = Helpers.randomAddress();
    }

    /**
     * Creates an instance of the interface which is backed by in-memory structures, only.
     */
    public TestingState(TestingBlock block) {
        this.dataStore = new MemoryBackedDataStore();
        IAccountStore premined = this.dataStore.createAccount(PREMINED_ADDRESS.toByteArray());
        premined.setBalance(PREMINED_AMOUNT);
        premined = this.dataStore.createAccount(BIG_PREMINED_ADDRESS.toByteArray());
        premined.setBalance(PREMINED_BIG_AMOUNT);
        this.blockDifficulty = block.getDifficulty();
        this.blockNumber = block.getNumber();
        this.blockTimestamp = block.getTimestamp();
        this.blockNrgLimit = block.getEnergyLimit();
        this.blockCoinbase = block.getCoinbase();
    }

    /**
     * Creates an instance of the interface which is backed by a directory on disk.
     * 
     * @param onDiskRoot The root directory which this implementation will use for persistence.
     * @param block The top block of the current state of this kernel.
     */
    public TestingState(File onDiskRoot, TestingBlock block) {
        this.dataStore = new DirectoryBackedDataStore(onDiskRoot);
        // Try to open the account, creating it if doesn't exist.
        IAccountStore premined = this.dataStore.openAccount(PREMINED_ADDRESS.toByteArray());
        if (null == premined) {
            premined = this.dataStore.createAccount(PREMINED_ADDRESS.toByteArray());
        }
        premined.setBalance(PREMINED_AMOUNT);
        this.blockDifficulty = block.getDifficulty();
        this.blockNumber = block.getNumber();
        this.blockTimestamp = block.getTimestamp();
        this.blockNrgLimit = block.getEnergyLimit();
        this.blockCoinbase = block.getCoinbase();
    }

    @Override
    public IExternalState newChildExternalState() {
        return new TransactionalState(this);
    }

    @Override
    public void commit() {
        throw new AssertionError("This class does not implement this method.");
    }

    @Override
    public void commitTo(IExternalState target) {
        throw new AssertionError("This class does not implement this method.");
    }

    @Override
    public byte[] getBlockHashByNumber(long blockNumber) {
        throw new AssertionError("No equivalent concept in the Avm.");
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
        lazyCreateAccount(address.toByteArray()).setCode(code);
    }

    @Override
    public byte[] getTransformedCode(AionAddress address) {
        return internalGetTransformedCode(address);
    }

    @Override
    public void setTransformedCode(AionAddress address, byte[] bytes) {
        lazyCreateAccount(address.toByteArray()).setTransformedCode(bytes);
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

    private IAccountStore lazyCreateAccount(byte[] address) {
        IAccountStore account = this.dataStore.openAccount(address);
        if (null == account) {
            account = this.dataStore.createAccount(address);
        }
        return account;
    }

    @Override
    public boolean destinationAddressIsSafeForThisVM(AionAddress address) {
        // This implementation knows about contract address prefixes (just used by tests - real kernel stores out-of-band meta-data).
        // So, it is valid to use any regular address or AVM contract address.
        byte[] code = internalGetTransformedCode(address);
        return (code == null) || (address.toByteArray()[0] == TestingState.AVM_CONTRACT_PREFIX);
    }

    @Override
    public long getBlockNumber() {
        return blockNumber;
    }

    @Override
    public long getBlockTimestamp() {
        return blockTimestamp;
    }

    @Override
    public long getBlockEnergyLimit() {
        return blockNrgLimit;
    }

    @Override
    public BigInteger getBlockDifficulty() {
        return blockDifficulty;
    }

    @Override
    public AionAddress getMinerAddress() {
        return new AionAddress(blockCoinbase.toByteArray());
    }

    @Override
    public void refundAccount(AionAddress address, BigInteger amount) {
        // This method may have special logic in the kernel. Here it is just adjustBalance.
        internalAdjustBalance(address, amount);
    }

    public void generateBlock() {
        this.blockNumber ++;
        this.blockTimestamp += blockTimeMillis;
    }

    private void internalAdjustBalance(AionAddress address, BigInteger delta) {
        IAccountStore account = lazyCreateAccount(address.toByteArray());
        BigInteger start = account.getBalance();
        account.setBalance(start.add(delta));
    }

    private byte[] internalGetTransformedCode(AionAddress address) {
        IAccountStore account = this.dataStore.openAccount(address.toByteArray());
        return (null != account)
                ? account.getTransformedCode()
                : null;
    }
}
