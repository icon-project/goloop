# ICON CONFIGURATION 

## Introduction
This document specifies ICON configurations.

## Attributes
|  Attribute                               | Simple Description                                               | Default value      |
|------------------------------------------|:-----------------------------------------------------------------|--------------------|
| termPeriod                               | number of blocks that forms a period                             | 43200              |
| mainPRepCount                            | number of maximum main PRep nodes                                | 22                 |
| subPRepCount                             | number of maximum sub PRep nodes                                 | 78                 |
| irep                                     | expected Monthly Reward per Representative                       | 0                  |
| rrep                                     | expected Monthly Reward per EEP                                  | 1200               |
| bondRequirement                          | percentage that requires bond to make delegation fully staked    | 5                  |
| unbondingMax                             | maximum unbonding slot per account                               | 10                 |
| unbondingPeriodMultiplier                | unbond lock period multiplier                                    | 7                  |
| unstakeSlotMax                           | maximum unstake slot per account                                 | 1000               |
| lockMinMultiplier                        | mininum unstake lock period term multiplier                      | 5                  |
| lockMaxMultiplier                        | maximum unstake lock period term multiplier                      | 20                 |
| validationPenaltyCondition               | consecutive validation fail count that imposes penalty on PRep   | 660                |
| consistentValidationPenaltyCondition     | cumulative validation fail count that imposes penalty on PRep    | 5                  |
| consistentValidationPenaltyMask          | number of opportunities of consistentValidationPenaltyCondition  | 30                 |
| consistentValidationPenaltySlashRatio    | percentage of bond slashed when it gets penalty                  | 10                 |
| rewardFund                               | IISS 3.1 Reward Fund Attributes                                  |                    |

## rewardFund Attributes for ICON2
|  Attribute                               | Simple Description                                                          | Default value      |
|------------------------------------------|:----------------------------------------------------------------------------|--------------------|
| Iglobal                                  | Iglobal is total amount of issuance on a term                               | 15552000000        |
| Iprep                                    | Iprep is percentage of Iglobal to calculate Representative Reward Fund      | 50                 |
| Icps                                     | Icps is percentage of Iglobal to calculate Contribution Reward Fund         | 0                  |
| Irelay                                   | Irelay is percentage of Iglobal to calculate Relayer Reward Fund            | 0                  |
| Ivoter                                   | Ivoter is percentage of Iglobal to calculate Delegation Reward Fund         | 50                 |

## Explanation
### termPeriod
Term is a cycle, on which system can measure contribution of users and calculate its corresponding reward. Its default
value is equivalence of a day, and a block takes 2 seconds on average.

### mainPRepCount, subPRepCount
PRep consists of mainPRep and subPRep. mainPRep can participate in validating and voting in the process of consensus. 
Whereas subPRep is registered as a PRep, but it can't participate in those activities. On a regular term basis, 
PRep is ordered by a specific formula(mostly bonded delegation) and its top 22 PRep is elected as a mainPRep, and
the bottom 78 PRep become a subPRep.

### irep, rrep
irep and rrep are variables that used in IISS 2.0 reward calculation.

### bondRequirement
Each account can participate in the system as a way of stake, delegation, and bond. However, the amount of whole delegation
of a user cannot be utilized without bond. The bondRequirement defines its percentage that requires bond to make delegation fully
utilized. Based on bondRequirement, the system calculates bondedDelegation(not delegation by itself) and uses it when it orders PReps.

### unbondingPeriodMultiplier
User can unbond its bonds, but it takes several lock period. unbondingPeriodMultiplier defines unbond lock period multiplier.
unbonding lock period = unbondingPeriodMultiplier * termPeriod

### unstakeSlotMax
User can unstake its staking, but it cannot do it unlimitedly. unstakeSlotMax maximum unstake slot per account

### lockMinMultiplier, lockMaxMultiplier
Unstaking takes a certain period of time just as unbonding. lockMinMultiplier defines mininum unstake lock period term multiplier
and lockMaxMultiplier defines maximum unstake lock period term multiplier. Unstake lock period can vary based on system
environment at a particular moment.
unstake lock period MIN = lockMinMultiplier * termPeriod
unstake lock period MAX = lockMaxMultiplier * termPeriod

### validationPenaltyCondition
Although mainPRep has a permission to validate and vote, it can get penalty when it fails to do those actions.
validationPenaltyCondition defines consecutive validation fail count that imposes penalty on PRep.

### consistentValidationPenaltyCondition, consistentValidationPenaltyMask
Along with validationPenaltyCondition, Sanction also can be imposed based on a cumulative count of failures of validating.
consistentValidationPenaltyCondition defines the cumulative validation fail count that imposes penalty on the PRep and
consistentValidationPenaltyMask defines the number of opportunities of it.

### consistentValidationPenaltySlashRatio
This defines percentage of bond slashed when it gets a penalty.

### rewardFund
rewardFund variables are newly introduced in ICON2. Please refer to the document of ICON2.

## Example
~~~json
{
    "termPeriod": 43200,
    "mainPRepCount": 22,
    "subPRepCount": 78,
    "irep": 0,
    "rrep": 1200,
    "bondRequirement": 5,
    "unbondingPeriodMultiplier": 7,
    "unstakeSlotMax": 1000,
    "lockMinMultiplier": 5,
    "lockMaxMultiplier": 20,
    "unbondingMax": 100,
    "validationPenaltyCondition": 660,
    "consistentValidationPenaltyCondition": 5,
    "consistentValidationPenaltyMask": 30,
    "consistentValidationPenaltySlashRatio": 10,
    "rewardFund": {
        "Iglobal": 15552000000,
        "Iprep": 50,
        "Icps": 0,
        "Irelay": 0,
        "Ivoter": 50
    }
}
~~~