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

import org.junit.jupiter.api.Assertions;

import java.util.List;
import java.util.Map;
import java.util.function.Consumer;
import java.util.stream.Collectors;

public  class Matcher implements Consumer<String> {
    public static class Item {
        private final String text;
        private final boolean expected;
        private boolean actual = false;

        public Item(String text, boolean expected) {
            this.text = text;
            this.expected = expected;
        }

        public String getText() {
            return text;
        }

        public boolean isExpected() {
            return expected;
        }

        public boolean isActual() {
            return actual;
        }

        public void setActual(boolean actual) {
            this.actual = actual;
        }

        public void assertOK() {
            Assertions.assertEquals(expected, actual,
                    "for match " + text);
        }
    }

    private final List<Item> items;

    public Matcher(Map<String, Boolean> items) {
        this.items = items.entrySet().stream()
                .map(e -> new Item(e.getKey(), e.getValue()))
                .collect(Collectors.toList());
    }

    public void accept(String msg) {
        for (var item : items) {
            if (msg.indexOf(item.getText()) > 0) {
                item.setActual(true);
            }
        }
    }

    public void assertOK() {
        items.forEach(Item::assertOK);
    }
}
