*Storyboard Usage Guide*

**Why We Use Storyboard**
Storyboard is a powerful tool for designing and visualizing user interfaces. It allows us to create a visual representation of the user interface and the flow between different screens in our application. This makes it easier to understand and communicate the design, and it also helps us to identify potential issues early in the development process.

For more information on why we use Storyboard, check out the [official Storybook documentation](https://storybook.js.org/docs/react/get-started/introduction).


**How to Start the Service**
To start the service, follow these steps:

```bash
cd webui
yarn install
yarn storybook
```

**How to Write a Story**
Writing a story in Storyboard involves creating a new file in the stories directory. Each story file should export a default function that defines the story. Here's a basic example:

```javascript
import React from 'react';
import MyComponent from '../components/MyComponent';

export default {
  title: 'MyComponent',
  component: MyComponent,
};

export const MyStory = () => <MyComponent />;
```

In this example, MyComponent is the component we want to create a story for. The MyStory function defines how the component should be rendered. You can create multiple stories for each component to show different states or variations of the component.

Remember, the goal of a story is to exercise the UI. Try to cover all the different states and variations of your component in your stories.

Happy story writing!