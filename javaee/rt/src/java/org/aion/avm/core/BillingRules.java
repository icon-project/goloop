package org.aion.avm.core;


/**
 * A container class which includes static routines to calculate more complex fees based on multiple factors.
 * This makes these easier to scrutinize, test, and update.
 */
public class BillingRules {
    /**
     * The minimum cost of ANY transaction, applied no matter whether it is a deployment or just a normal call (balance transfers are also calls).
     */
    public static final int BASIC_TRANSACTION_COST = 21_000;
    
    /**
     * The minimum cost of a deployment, applied on top of deployment expenses related to code size of class count.
     */
    public static final int DEPLOYMENT_BASE_COST = 200_000;
    
    /**
     * The cost, per byte, of a deployment (specifically, the JAR).
     */
    public static final int DEPLOYMENT_PER_BYTE_JAR_COST = 1;
    
    /**
     * The cost, per class, of a deployment.
     */
    public static final int DEPLOYMENT_PER_CLASS_COST = 1000;
    
    
    /**
     * The cost of deploying a contract is based on the number of classes (as they represent JVM resources) and the size of the code.
     * 
     * @param numberOfClassesProvided The number of classes defined in the jar.
     * @param sizeOfJarInBytes The physical size of the jar (its compressed size).
     * @return The total fee for such a deployment.
     */
    public static long getDeploymentFee(long numberOfClassesProvided, long sizeOfJarInBytes) {
        return 0L
        // All deployments have the base cost.
                + DEPLOYMENT_BASE_COST
        // A per-byte JAR code cost.
                + (sizeOfJarInBytes * DEPLOYMENT_PER_BYTE_JAR_COST)
        // A per-class code cost.
                + (numberOfClassesProvided * DEPLOYMENT_PER_CLASS_COST)
                ;
    }
    
    /**
     * The basic cost of a transaction including the given data.
     * 
     * @return The basic cost, in energy, of a transaction with this data.
     */
    public static long getBasicTransactionCost(byte[] transactionData) {
        int cost = BASIC_TRANSACTION_COST;
        for (byte b : transactionData) {
            cost += (b == 0) ? 4 : 64;
        }
        return cost;
    }
}
