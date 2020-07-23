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

package foundation.icon.icx;

import foundation.icon.icx.data.Address;

/**
 * Wallet class signs the message(a transaction message to send)
 * using own key-pair
 */
public interface Wallet {

    /**
     * Gets the address corresponding the key of the wallet
     *
     * @return address
     */
    Address getAddress();

    /**
     * Signs the data to generate a signature
     *
     * @param data to sign
     * @return signature
     */
    byte[] sign(byte[] data);
}
