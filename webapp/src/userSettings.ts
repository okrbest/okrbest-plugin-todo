// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export enum UserSettingKey {
    Language = 'todo-plugin-language',
}

export class UserSettings {
    static get(key: UserSettingKey): string | null {
        return localStorage.getItem(key);
    }

    static set(key: UserSettingKey, value: string | null): void {
        if (!Object.values(UserSettingKey).includes(key)) {
            return;
        }
        if (value === null) {
            localStorage.removeItem(key);
        } else {
            localStorage.setItem(key, value);
        }
    }

    static get language(): string | null {
        return UserSettings.get(UserSettingKey.Language);
    }

    static set language(newValue: string | null) {
        UserSettings.set(UserSettingKey.Language, newValue);
    }
}
