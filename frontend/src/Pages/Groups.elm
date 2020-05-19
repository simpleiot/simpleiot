module Pages.Groups exposing (Flags, Model, Msg, page)

import Components.Form as Form
import Components.Icon as Icon
import Data.Device as D
import Data.Group as O
import Data.User as U
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Global
import List.Extra
import Page exposing (Document, Page)
import Utils.Styles exposing (palette, size)


type alias Flags =
    ()


type alias Model =
    { groupEdit : Maybe O.Group
    , newUser : Maybe NewUser
    , newDevice : Maybe NewDevice
    }


empty : Model
empty =
    { groupEdit = Nothing
    , newUser = Nothing
    , newDevice = Nothing
    }


type alias NewUser =
    { groupId : String
    , userEmail : String
    }


type alias NewDevice =
    { groupId : String
    , deviceId : String
    }


type Msg
    = PostGroup O.Group
    | EditGroup O.Group
    | DiscardGroupEdits
    | NewGroup
    | DeleteGroup String
    | RemoveUser O.Group String
    | AddUser String
    | CancelAddUser
    | EditNewUser String
    | SaveNewUser O.Group String
    | RemoveDevice String String
    | AddDevice String
    | CancelAddDevice
    | EditNewDevice String
    | SaveNewDevice String String


page : Page Flags Model Msg
page =
    Page.component
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }


init : Global.Model -> Flags -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ _ =
    ( empty
    , Cmd.none
    , Cmd.batch
        [ Global.send Global.RequestGroups
        , Global.send Global.RequestUsers
        , Global.send Global.RequestDevices
        ]
    )


update : Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update global msg model =
    case global.auth of
        Global.SignedOut _ ->
            ( model, Cmd.none, Cmd.none )

        Global.SignedIn sess ->
            case msg of
                EditGroup group ->
                    ( { model | groupEdit = Just group }
                    , Cmd.none
                    , Cmd.none
                    )

                DiscardGroupEdits ->
                    ( { model | groupEdit = Nothing }
                    , Cmd.none
                    , Cmd.none
                    )

                PostGroup group ->
                    ( { model | groupEdit = Nothing }
                    , Cmd.none
                    , Global.send <| Global.UpdateGroup group
                    )

                NewGroup ->
                    ( { model | groupEdit = Just O.empty }
                    , Cmd.none
                    , Cmd.none
                    )

                DeleteGroup id ->
                    ( { model | groupEdit = Nothing }
                    , Cmd.none
                    , Global.send <| Global.DeleteGroup id
                    )

                RemoveUser group userId ->
                    let
                        users =
                            List.filter
                                (\ur -> ur.userId /= userId)
                                group.users

                        updatedGroup =
                            { group | users = users }
                    in
                    ( model
                    , Cmd.none
                    , Global.send <| Global.UpdateGroup updatedGroup
                    )

                AddUser groupId ->
                    ( { model | newUser = Just { groupId = groupId, userEmail = "" } }
                    , Cmd.none
                    , Cmd.none
                    )

                CancelAddUser ->
                    ( { model | newUser = Nothing }
                    , Cmd.none
                    , Cmd.none
                    )

                EditNewUser userEmail ->
                    case model.newUser of
                        Just newUser ->
                            ( { model | newUser = Just { newUser | userEmail = userEmail } }
                            , Cmd.none
                            , Global.send <| Global.CheckUser userEmail
                            )

                        Nothing ->
                            ( model, Cmd.none, Cmd.none )

                SaveNewUser group userId ->
                    let
                        -- only add user if it does not already exist
                        users =
                            case
                                List.Extra.find
                                    (\ur -> ur.userId == userId)
                                    group.users
                            of
                                Just _ ->
                                    group.users

                                Nothing ->
                                    { userId = userId, roles = [ "user" ] } :: group.users

                        updatedGroup =
                            { group | users = users }
                    in
                    ( { model | newUser = Nothing }
                    , Cmd.none
                    , Global.send <| Global.UpdateGroup updatedGroup
                    )

                RemoveDevice groupId deviceId ->
                    ( model
                    , Cmd.none
                    , case
                        List.Extra.find (\d -> d.id == deviceId)
                            sess.data.devices
                      of
                        Just device ->
                            let
                                groups =
                                    List.filter (\o -> o /= groupId)
                                        device.groups
                            in
                            Global.send <|
                                Global.UpdateDeviceGroups device.id groups

                        Nothing ->
                            Cmd.none
                    )

                AddDevice groupId ->
                    ( { model | newDevice = Just { groupId = groupId, deviceId = "" } }
                    , Cmd.none
                    , Cmd.none
                    )

                CancelAddDevice ->
                    ( { model | newDevice = Nothing }
                    , Cmd.none
                    , Cmd.none
                    )

                EditNewDevice deviceId ->
                    case model.newDevice of
                        Just newDevice ->
                            ( { model | newDevice = Just { newDevice | deviceId = deviceId } }
                            , Cmd.none
                            , Global.send <| Global.CheckDevice deviceId
                            )

                        Nothing ->
                            ( model, Cmd.none, Cmd.none )

                SaveNewDevice groupId deviceId ->
                    ( { model | newDevice = Nothing }
                    , Cmd.none
                    , case
                        List.Extra.find (\d -> d.id == deviceId)
                            sess.data.devices
                      of
                        Just device ->
                            let
                                groups =
                                    case
                                        List.Extra.find (\o -> o == groupId)
                                            device.groups
                                    of
                                        Just _ ->
                                            device.groups

                                        Nothing ->
                                            groupId :: device.groups
                            in
                            Global.send <|
                                Global.UpdateDeviceGroups device.id groups

                        Nothing ->
                            Cmd.none
                    )


subscriptions : Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.none


view : Global.Model -> Model -> Document Msg
view global model =
    { title = "Groups"
    , body =
        [ case global.auth of
            Global.SignedIn sess ->
                column
                    [ width fill, spacing 32 ]
                    [ el [ padding 16, Font.size 24 ] <| text "Groups"
                    , el [ padding 16, width fill, Font.bold ] <| Form.button "new group" palette.green NewGroup
                    , viewGroups sess model
                    ]

            _ ->
                el [ padding 16 ] <| text "Sign in to view your groups."
        ]
    }


viewGroups : Global.Session -> Model -> Element Msg
viewGroups sess model =
    column
        [ width fill
        , spacing 40
        ]
    <|
        List.map (\o -> viewGroup sess model o.mod o.group) <|
            mergeGroupEdit sess.data.groups model.groupEdit


type alias GroupMod =
    { group : O.Group
    , mod : Bool
    }


mergeGroupEdit : List O.Group -> Maybe O.Group -> List GroupMod
mergeGroupEdit groups groupEdit =
    case groupEdit of
        Just edit ->
            let
                groupsMapped =
                    List.map
                        (\o ->
                            if edit.id == o.id then
                                { group = edit, mod = True }

                            else
                                { group = o, mod = False }
                        )
                        groups
            in
            if edit.id == "" then
                [ { group = edit, mod = True } ] ++ groupsMapped

            else
                groupsMapped

        Nothing ->
            List.map (\o -> { group = o, mod = False }) groups


viewGroup : Global.Session -> Model -> Bool -> O.Group -> Element Msg
viewGroup sess model modded group =
    let
        devices =
            List.filter
                (\d ->
                    case List.Extra.find (\groupId -> group.id == groupId) d.groups of
                        Just _ ->
                            True

                        Nothing ->
                            False
                )
                sess.data.devices
    in
    column
        ([ width fill
         , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
         , Border.color palette.black
         , spacing 6
         ]
            ++ (if modded then
                    [ Background.color palette.orange
                    , below <|
                        Form.buttonRow
                            [ Form.button "discard" palette.pale <| DiscardGroupEdits
                            , Form.button "save" palette.green <| PostGroup group
                            ]
                    ]

                else
                    []
               )
        )
        [ if group.id == "00000000-0000-0000-0000-000000000000" then
            el [ padding 16 ] (text group.name)

          else
            row
                []
                [ Form.viewTextProperty
                    { name = "Groupanization name"
                    , value = group.name
                    , action = \x -> EditGroup { group | name = x }
                    }
                , Icon.x (DeleteGroup group.id)
                ]
        , row []
            [ el [ padding 16, Font.italic, Font.color palette.gray ] <| text "Users"
            , case model.newUser of
                Just newUser ->
                    if newUser.groupId == group.id then
                        Icon.userX CancelAddUser

                    else
                        Icon.userPlus (AddUser group.id)

                Nothing ->
                    Icon.userPlus (AddUser group.id)
            ]
        , case model.newUser of
            Just ua ->
                if ua.groupId == group.id then
                    row []
                        [ Form.viewTextProperty
                            { name = "Enter new user email address"
                            , value = ua.userEmail
                            , action = \x -> EditNewUser x
                            }
                        , case sess.newGroupUser of
                            Just user ->
                                Icon.userPlus (SaveNewUser group user.id)

                            Nothing ->
                                Element.none
                        ]

                else
                    Element.none

            Nothing ->
                Element.none
        , viewUsers group sess.data.users
        , row []
            [ el [ padding 16, Font.italic, Font.color palette.gray ] <| text "Devices"
            , case model.newDevice of
                Just newDevice ->
                    if newDevice.groupId == group.id then
                        Icon.x CancelAddDevice

                    else
                        Icon.plus (AddDevice group.id)

                Nothing ->
                    Icon.plus (AddDevice group.id)
            ]
        , case model.newDevice of
            Just nd ->
                if nd.groupId == group.id then
                    row []
                        [ Form.viewTextProperty
                            { name = "Enter new device ID"
                            , value = nd.deviceId
                            , action = \x -> EditNewDevice x
                            }
                        , case sess.newGroupDevice of
                            Just dev ->
                                Icon.plus (SaveNewDevice group.id dev.id)

                            Nothing ->
                                Element.none
                        ]

                else
                    Element.none

            Nothing ->
                Element.none
        , viewDevices group devices
        ]


viewUsers : O.Group -> List U.User -> Element Msg
viewUsers group users =
    column [ spacing 6, paddingEach { top = 0, right = 16, bottom = 0, left = 32 } ]
        (List.map
            (\ur ->
                case List.Extra.find (\u -> u.id == ur.userId) users of
                    Just user ->
                        row [ padding 16 ]
                            [ text
                                (user.first
                                    ++ " "
                                    ++ user.last
                                    ++ " <"
                                    ++ user.email
                                    ++ ">"
                                )
                            , Icon.userX (RemoveUser group user.id)
                            ]

                    Nothing ->
                        el [ padding 16 ] <| text "User not found"
            )
            group.users
        )


viewDevices : O.Group -> List D.Device -> Element Msg
viewDevices group devices =
    column [ spacing 6, paddingEach { top = 0, right = 16, bottom = 0, left = 32 } ]
        (List.map
            (\d ->
                row [ padding 16 ]
                    [ text
                        ("("
                            ++ d.id
                            ++ ") "
                            ++ d.config.description
                        )
                    , Icon.x (RemoveDevice group.id d.id)
                    ]
            )
            devices
        )
