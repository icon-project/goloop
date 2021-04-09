# ICON CONFIGURATION 

## Introduction
This document specifies ICON configurations.

## Attributes
|  Attribute                               | Simple Description                                               | Default value      |
|------------------------------------------|:-----------------------------------------------------------------|--------------------|
| termPeriod                               | number of blocks that consists of a period                       | 43120              |
| mainPRepCount                            | number of maximum main PRep nodes                                | 22                 |
| subPRepCount                             | number of maximum sub PRep nodes                                 | 78                 |
| irep                                     | Expected Monthly Reward per Representative                       | 0                  |
| rrep                                     | Expected Monthly Reward per EEP                                  | 1200               |
| bondRequirement                          | Percentage that requires bond to make delegation fully staked    | 5                  |
| unbondingPeriodMultiplier                | unbond lock period multiplier                                    | 7                  |
| unstakeSlotMax                           | maximum unstake slot per account                                 | 100                |
| lockMinMultiplier                        | mininum unstake lock period term multiplier                      | 5                  |
| lockMaxMultiplier                        | maximum unstake lock period term multiplier                      | 20                 |
| validationPenaltyCondition               | consecutive validation fail count that imposes penalty on PRep   | 660                |
| consistentValidationPenaltyCondition     | cumulative validation fail count that imposes penalty on PRep    | 5                  |
| consistentValidationPenaltyMask          | number of opportunities of consistentValidationPenaltyCondition  | 30                 |
| consistentValidationPenaltySlashRatio    | Percentage of bond slashed when it gets penalty                  | 10                 |

## rewardFund Attributes for ICON2
|  Attribute                               | Simple Description                                               | Default value      |
|------------------------------------------|:-----------------------------------------------------------------|--------------------|
| Iglobal                                  | Iglobal is multiplier, providing reward fund                     | YearBlock * IScoreICXRatio|
| Iprep                                    | Iprep is multiplier of Representative Reward Fund(with Iglobal)  | 50                 |
| Icps                                     | Icps is multiplier of Contribution Reward Fund(with Iglobal)     | 0                  |
| Irelay                                   | Irelay is multiplier of Relayer Reward Fund(with Iglobal)        | 0                  |
| Ivoter                                   | Ivoter is multiplier of Delegation Reward Fund(with Iglobal)     | 50                 |

## explanation
### termPeriod
Term is ~~


## example
~~~json
{
"termPeriod": 43120,
"mainPRepCount": 22,
"subPRepCount": 78,
"irep": 0,
"rrep": 1200,
"bondRequirement": 5,
"unbondingPeriodMultiplier": 7,
"unstakeSlotMax": 100,
"lockMinMultiplier": 5,
"lockMaxMultiplier": 20,
"rewardFund": {
"Iglobal": 15552000000,
"Iprep": 50,
"Icps": 0,
"Irelay": 0,
"Ivoter": 50
},
"validationPenaltyCondition": 660,
"consistentValidationPenaltyCondition": 5,
"consistentValidationPenaltyMask": 30,
"consistentValidationPenaltySlashRatio": 10
}
~~~