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
    path: '/exportTable',
    name: "export-table",
    icon: "smile",
    component: "./exportTable",
  },
  {
    path: '/',
    redirect: '/debug',
  },
  {
    component: './404',
  },
];
