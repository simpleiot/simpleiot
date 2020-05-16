module Pages.Orgs exposing (Model, Msg, page)

import Components.Form as Form
import Components.Icon as Icon
import Data.Device as D
import Data.Org as O
import Data.User as U
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Generated.Params as Params
import Global
import List.Extra
import Spa.Page
import Spa.Types as Types
import Utils.Spa exposing (Page)
import Utils.Styles exposing (palette, size)


page : Page Params.Orgs Model Msg model msg appMsg
page =
    Spa.Page.component
        { title = always "Orgs"
        , init = init
        , update = update
        , subscriptions = always subscriptions
        , view = view
        }



-- INIT


type alias Model =
    { orgEdit : Maybe O.Org
    , newUser : Maybe NewUser
    , newDevice : Maybe NewDevice
    }


empty : Model
empty =
    { orgEdit = Nothing
    , newUser = Nothing
    , newDevice = Nothing
    }


type alias NewUser =
    { orgId : String
    , userEmail : String
    }


type alias NewDevice =
    { orgId : String
    , deviceId : String
    }


init : Types.PageContext route Global.Model -> Params.Orgs -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ _ =
    ( empty
    , Cmd.none
    , Cmd.batch
        [ Spa.Page.send Global.RequestOrgs
        , Spa.Page.send Global.RequestUsers
        , Spa.Page.send Global.RequestDevices
        ]
    )



-- UPDATE


type Msg
    = PostOrg O.Org
    | EditOrg O.Org
    | DiscardOrgEdits
    | NewOrg
    | RemoveUser O.Org String
    | AddUser String
    | CancelAddUser
    | EditNewUser String
    | SaveNewUser O.Org String
    | RemoveDevice String String
    | AddDevice String
    | CancelAddDevice
    | EditNewDevice String
    | SaveNewDevice String String


update : Types.PageContext route Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update context msg model =
    case context.global of
        Global.SignedOut _ ->
            ( model, Cmd.none, Cmd.none )

        Global.SignedIn sess ->
            case msg of
                EditOrg org ->
                    ( { model | orgEdit = Just org }
                    , Cmd.none
                    , Cmd.none
                    )

                DiscardOrgEdits ->
                    ( { model | orgEdit = Nothing }
                    , Cmd.none
                    , Cmd.none
                    )

                PostOrg org ->
                    ( { model | orgEdit = Nothing }
                    , Cmd.none
                    , Spa.Page.send <| Global.UpdateOrg org
                    )

                NewOrg ->
                    ( { model | orgEdit = Just O.empty }
                    , Cmd.none
                    , Cmd.none
                    )

                RemoveUser org userId ->
                    let
                        users =
                            List.filter
                                (\ur -> ur.userId /= userId)
                                org.users

                        updatedOrg =
                            { org | users = users }
                    in
                    ( model
                    , Cmd.none
                    , Spa.Page.send <| Global.UpdateOrg updatedOrg
                    )

                AddUser orgId ->
                    ( { model | newUser = Just { orgId = orgId, userEmail = "" } }
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
                            , Spa.Page.send <| Global.CheckUser userEmail
                            )

                        Nothing ->
                            ( model, Cmd.none, Cmd.none )

                SaveNewUser org userId ->
                    let
                        -- only add user if it does not already exist
                        users =
                            case
                                List.Extra.find
                                    (\ur -> ur.userId == userId)
                                    org.users
                            of
                                Just _ ->
                                    org.users

                                Nothing ->
                                    { userId = userId, roles = [ "user" ] } :: org.users

                        updatedOrg =
                            { org | users = users }
                    in
                    ( { model | newUser = Nothing }
                    , Cmd.none
                    , Spa.Page.send <| Global.UpdateOrg updatedOrg
                    )

                RemoveDevice orgId deviceId ->
                    ( model
                    , Cmd.none
                    , case
                        List.Extra.find (\d -> d.id == deviceId)
                            sess.data.devices
                      of
                        Just device ->
                            let
                                orgs =
                                    List.filter (\o -> o /= orgId)
                                        device.orgs
                            in
                            Spa.Page.send <|
                                Global.UpdateDeviceOrgs device.id orgs

                        Nothing ->
                            Cmd.none
                    )

                AddDevice orgId ->
                    ( { model | newDevice = Just { orgId = orgId, deviceId = "" } }
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
                            , Spa.Page.send <| Global.CheckDevice deviceId
                            )

                        Nothing ->
                            ( model, Cmd.none, Cmd.none )

                SaveNewDevice orgId deviceId ->
                    ( { model | newDevice = Nothing }
                    , Cmd.none
                    , case
                        List.Extra.find (\d -> d.id == deviceId)
                            sess.data.devices
                      of
                        Just device ->
                            let
                                orgs =
                                    case
                                        List.Extra.find (\o -> o == orgId)
                                            device.orgs
                                    of
                                        Just _ ->
                                            device.orgs

                                        Nothing ->
                                            orgId :: device.orgs
                            in
                            Spa.Page.send <|
                                Global.UpdateDeviceOrgs device.id orgs

                        Nothing ->
                            Cmd.none
                    )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none



-- VIEW


view : Types.PageContext route Global.Model -> Model -> Element Msg
view context model =
    case context.global of
        Global.SignedIn sess ->
            column
                [ width fill, spacing 32 ]
                [ el [ padding 16, Font.size 24 ] <| text "Orgs"
                , el [ padding 16, width fill, Font.bold ] <| Form.button "new organization" palette.green NewOrg
                , viewOrgs sess model
                ]

        _ ->
            el [ padding 16 ] <| text "Sign in to view your orgs."


viewOrgs : Global.Session -> Model -> Element Msg
viewOrgs sess model =
    column
        [ width fill
        , spacing 40
        ]
    <|
        List.map (\o -> viewOrg sess model o.mod o.org) <|
            mergeOrgEdit sess.data.orgs model.orgEdit


type alias OrgMod =
    { org : O.Org
    , mod : Bool
    }


mergeOrgEdit : List O.Org -> Maybe O.Org -> List OrgMod
mergeOrgEdit orgs orgEdit =
    case orgEdit of
        Just edit ->
            let
                orgsMapped =
                    List.map
                        (\o ->
                            if edit.id == o.id then
                                { org = edit, mod = True }

                            else
                                { org = o, mod = False }
                        )
                        orgs
            in
            if edit.id == "" then
                [ { org = edit, mod = True } ] ++ orgsMapped

            else
                orgsMapped

        Nothing ->
            List.map (\o -> { org = o, mod = False }) orgs


viewOrg : Global.Session -> Model -> Bool -> O.Org -> Element Msg
viewOrg sess model modded org =
    let
        devices =
            List.filter
                (\d ->
                    case List.Extra.find (\orgId -> org.id == orgId) d.orgs of
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
                            [ Form.button "discard" palette.pale <| DiscardOrgEdits
                            , Form.button "save" palette.green <| PostOrg org
                            ]
                    ]

                else
                    []
               )
        )
        [ Form.viewTextProperty
            { name = "Organization name"
            , value = org.name
            , action = \x -> EditOrg { org | name = x }
            }
        , row []
            [ el [ padding 16, Font.italic, Font.color palette.gray ] <| text "Users"
            , case model.newUser of
                Just newUser ->
                    if newUser.orgId == org.id then
                        Icon.userX CancelAddUser

                    else
                        Icon.userPlus (AddUser org.id)

                Nothing ->
                    Icon.userPlus (AddUser org.id)
            ]
        , case model.newUser of
            Just ua ->
                if ua.orgId == org.id then
                    row []
                        [ Form.viewTextProperty
                            { name = "Enter new user email address"
                            , value = ua.userEmail
                            , action = \x -> EditNewUser x
                            }
                        , case sess.newOrgUser of
                            Just user ->
                                Icon.userPlus (SaveNewUser org user.id)

                            Nothing ->
                                Element.none
                        ]

                else
                    Element.none

            Nothing ->
                Element.none
        , viewUsers org sess.data.users
        , row []
            [ el [ padding 16, Font.italic, Font.color palette.gray ] <| text "Devices"
            , case model.newDevice of
                Just newDevice ->
                    if newDevice.orgId == org.id then
                        Icon.x CancelAddDevice

                    else
                        Icon.plus (AddDevice org.id)

                Nothing ->
                    Icon.plus (AddDevice org.id)
            ]
        , case model.newDevice of
            Just nd ->
                if nd.orgId == org.id then
                    row []
                        [ Form.viewTextProperty
                            { name = "Enter new device ID"
                            , value = nd.deviceId
                            , action = \x -> EditNewDevice x
                            }
                        , case sess.newOrgDevice of
                            Just dev ->
                                Icon.userPlus (SaveNewDevice org.id dev.id)

                            Nothing ->
                                Element.none
                        ]

                else
                    Element.none

            Nothing ->
                Element.none
        , viewDevices org devices
        ]


viewUsers : O.Org -> List U.User -> Element Msg
viewUsers org users =
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
                            , Icon.userX (RemoveUser org user.id)
                            ]

                    Nothing ->
                        el [ padding 16 ] <| text "User not found"
            )
            org.users
        )


viewDevices : O.Org -> List D.Device -> Element Msg
viewDevices org devices =
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
                    , Icon.x (RemoveDevice org.id d.id)
                    ]
            )
            devices
        )
