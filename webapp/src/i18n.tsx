// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import messages_en from '../i18n/en.json';
import messages_ko from '../i18n/ko.json';

export function getMessages(lang: string): {[key: string]: string} {
    switch (lang) {
    case 'ko':
        return messages_ko;
    case 'en':
    default:
        return messages_en;
    }
}
