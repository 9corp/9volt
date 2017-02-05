import React from 'react';
import {connect} from 'react-redux';
import {Header,List, Grid, Image} from 'semantic-ui-react'
import Image9 from '../assets/9volt-image.jpg'

const HomeView = () => {
    return (
        <div>
            <Grid>
                <Grid.Column width={10}>
                    <Header as='h1'>Welcome to 9volt</Header>
                    <p>
                    While there are a bunch of solutions for monitoring and alerting 
                    using time series data, there aren't many (or any?) modern solutions 
                    for 'regular'/'old-skool' remote monitoring similar to Nagios and Icinga.
                    </p>

                    <Header as="h2">9volt offers the following things out of the box:</Header>

                    <List bulleted>
                        <List.Item>Single binary deploy</List.Item>
                        <List.Item>Fully H/A</List.Item>
                        <List.Item>Incredibly easy to scale to hundreds of thousands of checks</List.Item>
                        <List.Item>Uses etcd for all configuration</List.Item>
                        <List.Item>Real-time configuration pick-up (update etcd - 9volt immediately picks up the change)</List.Item>
                        <List.Item>Interval based monitoring (ie. run check XYZ every 1s or 1y or 1d or even 1ms)</List.Item>
                        <List.Item>Natively supported monitors:
                            <List.List>
                                <List.Item>TCP</List.Item>
                                <List.Item>HTTP</List.Item>
                                <List.Item>Exec</List.Item>
                            </List.List>
                        </List.Item>
                        <List.Item>Natively supported alerters:
                            <List.List>
                                <List.Item>Slack</List.Item>
                                <List.Item>Pagerduty</List.Item>
                                <List.Item>Email</List.Item>
                            </List.List>
                        </List.Item>

                        <List.Item>RESTful API for querying current monitoring state and loaded configuration</List.Item>
                        <List.Item>Comes bundled with a web app for a quick visual view of the cluster (this-app)</List.Item>
                        <List.Item>Comes bundled with a binary tool to push parse YAML based configs and push them to etcd</List.Item>
                    </List>
                </Grid.Column>
                <Grid.Column width={6}>
                    <Image src={Image9} shape='rounded'/>
                </Grid.Column>
            </Grid>
           

        </div>
    );
};

export default connect()(HomeView)
