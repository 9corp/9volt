import React from 'react';
import {connect} from 'react-redux'
import {Link} from 'react-router';
import {push} from 'react-router-redux';

class Navigation extends React.Component {

  render() {
    return (
      <div>
          Navigation
      </div>
    );
  }
}

export default connect()(Navigation)
