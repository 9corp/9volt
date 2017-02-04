import React from 'react';
import {connect} from 'react-redux'
import {Link} from 'react-router';
import {push} from 'react-router-redux';

import { Menu, Icon, Sidebar, Image, Header, Segment } from 'semantic-ui-react'

const items = [
  {key:"home",icon:"home",title:"9-Volt",path:"/ui"},
  {key:"status",icon:"bar chart",title:"Status",path:"/ui/Status"},
  {key:"settings",icon:"settings",title:"Settings",path:"/ui/Settings"}
];

class NavBar extends React.Component {

  pushPath = (dispatch,path,name) => {
      dispatch(push(path));
  };

  render() {
    const {dispatch,currentRoute} = this.props;

    return (
      <Menu color='teal' icon='labeled' vertical inverted fixed="left">
        { items.map(i => {
            const {key, icon, title, path} = i;
            return (
              <Menu.Item key={key} name={key} active={currentRoute === path} onClick={() => this.pushPath(dispatch,path,key)}>
                <Icon name={icon} />
                {title}
              </Menu.Item>
            );
          })
        }
      </Menu>     
    );
  }
}

const mapStateToProps = (state) => ({
  currentRoute: state.routing.locationBeforeTransitions.pathname
});


export default connect(mapStateToProps)(NavBar)
