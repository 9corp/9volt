import React from 'react';
import ReactDOM from 'react-dom';
import {connect} from 'react-redux';

const HomeView = () => {
    return (
        <div>
            Welcome to 9volt!
        </div>
    );
};

export default connect()(HomeView)
