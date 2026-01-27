import React from 'react';
import PropTypes from 'prop-types';
import {useIntl} from 'react-intl';

const BumpButton = (props) => {
    const intl = useIntl();

    return (
        <button
            className='btn btn-primary'
            onClick={() => props.bump(props.issueId)}
        >{intl.formatMessage({id: 'Button.bump', defaultMessage: 'Bump'})}</button>
    );
};

BumpButton.propTypes = {
    issueId: PropTypes.string.isRequired,
    bump: PropTypes.func.isRequired,
};

export default BumpButton;
