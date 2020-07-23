/*
 * Copyright 2018 ICON Foundation
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

package foundation.icon.icx.data;

import java.math.BigDecimal;
import java.math.BigInteger;

public class IconAmount {

    private final BigDecimal value;
    private final int digit;

    public IconAmount(BigDecimal value, int digit) {
        this.value = value;
        this.digit = digit;
    }

    public int getDigit() {
        return digit;
    }

    @Override
    public String toString() {
        return value.toString();
    }

    public BigInteger asInteger() {
        return value.toBigInteger();
    }

    public BigDecimal asDecimal() {
        return value;
    }

    public BigInteger toLoop() {
        return value.multiply(getTenDigit(digit)).toBigInteger();
    }

    public IconAmount convertUnit(Unit unit) {
        BigInteger loop = toLoop();
        return IconAmount.of(new BigDecimal(loop).divide(getTenDigit(unit.getValue())), unit);
    }

    public IconAmount convertUnit(int digit) {
        BigInteger loop = toLoop();
        return IconAmount.of(new BigDecimal(loop).divide(getTenDigit(digit)), digit);
    }

    private BigDecimal getTenDigit(int digit) {
        return BigDecimal.TEN.pow(digit);
    }

    public enum Unit {
        LOOP(0),
        ICX(18);

        int digit;

        Unit(int digit) {
            this.digit = digit;
        }

        public int getValue() {
            return digit;
        }
    }

    public static IconAmount of(BigDecimal loop, int digit) {
        return new IconAmount(loop, digit);
    }

    public static IconAmount of(BigDecimal loop, Unit unit) {
        return of(loop, unit.getValue());
    }

    public static IconAmount of(String loop, int digit) {
        return of(new BigDecimal(loop), digit);
    }

    public static IconAmount of(String loop, Unit unit) {
        return of(new BigDecimal(loop), unit.getValue());
    }

    public static IconAmount of(BigInteger loop, int digit) {
        return of(new BigDecimal(loop), digit);
    }

    public static IconAmount of(BigInteger loop, Unit unit) {
        return of(new BigDecimal(loop), unit.getValue());
    }
}
