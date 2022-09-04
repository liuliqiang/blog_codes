'use strict';

class DivText extends React.Component {
    render() {
        return React.createElement(
            'div',
            {},
            this.props.text
        )
    }
}

class LikeButton extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            liked: false,
            text: "0",
            value: 0,
        };
    }

    render() {
        return React.createElement(
            'button',
            {
                onClick: ()=>{
                    let val = this.state.value + 1;
                    this.setState({
                        liked: true,
                        value: val,
                        text: val.toString(),
                    });
                }
            },
            React.createElement(DivText, {text: this.state.text})
        )
    }
}
