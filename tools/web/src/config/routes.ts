export default [
  {
    path: '/debug',
    name: 'debug',
    icon: 'smile',
    component: './debug',
  },
  {
    path: '/gm',
    name: 'gm',
    icon: "smile",
    component: "./gm",
  },
  {
    path: '/',
    redirect: '/debug',
  },
  {
    component: './404',
  },
];
