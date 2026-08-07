# Users/Groups

Users and Groups can be configured at any place in the node tree. The way
permissions work is users have access to the parent node and the parent nodes
children. In the below example, `Joe` has access to the `SBC` device because
both `Joe` and `SBC` are members of the `Site 1` group. `Joe` does have access
to the `root` node.

![group user](images/group-user.png)

If `Joe` logs in, the following view will be presented:

![joe nodes](images/joe-nodes.png)

## Schema

The configuration of a group and a user in it:

```yaml
nodes:
  - group:
      description: Site 1
  - user:
      parent: Site 1
      email: joe@example.com
      firstName: Joe
      lastName: Smith
      pass: his-password
      phone: "+12155551212"
      edgePoints:
        role: admin
```

A group carries a description and nothing else. Its place in the tree is what
gives it meaning, and the users and devices below it are what it groups.

A user is the one node type with no description, so a file finds it by `email`,
and by name when there is no email. `phone` is written as text so the leading
`+` is kept.

`role` is `admin` or `user` and lives under `edgePoints` rather than with the
points, because a role belongs to the connection between the user and the node
above rather than to the user. The same user mirrored into two places can hold
a different role in each.

An export carries `pass` as it was entered, so treat a file that contains user
nodes the way you would treat the passwords in it.
