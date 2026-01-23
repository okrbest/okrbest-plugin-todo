// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import messages_en from '../i18n/en.json';
import messages_ko from '../i18n/ko.json';

import {UserSettings} from './userSettings';

const supportedLanguages = ['en', 'ko'];

export function getMessages(lang: string): {[key: string]: string} {
    switch (lang) {
    case 'ko':
        return messages_ko;
    case 'en':
    default:
        return messages_en;
    }
}

export function getCurrentLanguage(): string {
    let lang = UserSettings.language;
    if (!lang) {
        if (supportedLanguages.includes(navigator.language)) {
            lang = navigator.language;
        } else if (supportedLanguages.includes(navigator.language.split(/[-_]/)[0])) {
            lang = navigator.language.split(/[-_]/)[0];
        } else {
            lang = 'ko'; // 기본 언어: 한국어
        }
    }
    return lang;
}

export function storeLanguage(lang: string): void {
    UserSettings.language = lang;
}
