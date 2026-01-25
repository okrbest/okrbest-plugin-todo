import React from 'react';
import PropTypes from 'prop-types';
import {useIntl} from 'react-intl';

import Button from 'src/widget/buttons/button';

const AcceptButton = (props) => {
    const intl = useIntl();

    return (
        <Button
            emphasis={'secondary'}
            onClick={() => props.accept(props.issueId)}
        >
            {intl.formatMessage({id: 'Button.accept', defaultMessage: 'Add to my list'})}
        </Button>
    );
};

AcceptButton.propTypes = {
    issueId: PropTypes.string.isRequired,
    accept: PropTypes.func.isRequired,
};

export default AcceptButton;
