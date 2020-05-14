module Pages.Orgs exposing (Model, Msg, page)

import Components.Button as Button
import Components.Form as Form
import Data.Data as Data
import Data.Device as D
import Data.Org as O
import Data.User as U
import Dict exposing (Dict)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
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
    { orgEdit : Maybe O.Org }


empty : Model
empty =
    { orgEdit = Nothing }


init : Types.PageContext route Global.Model -> Params.Orgs -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ _ =
    ( empty
    , Cmd.none
    , Cmd.batch [ Spa.Page.send Global.RequestOrgs, Spa.Page.send Global.RequestUsers ]
    )



-- UPDATE


type Msg
    = PostOrg O.Org
    | EditOrg O.Org
    | DiscardOrgEdits
    | NewOrg


update : Types.PageContext route Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update context msg model =
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
            , case context.global of
                Global.SignedIn _ ->
                    Spa.Page.send <| Global.UpdateOrg org

                Global.SignedOut _ ->
                    Cmd.none
            )

        NewOrg ->
            ( { model | orgEdit = Just O.empty }
            , Cmd.none
            , Cmd.none
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
                , el [ padding 16, width fill, Font.bold ] <| Button.view2 "new organization" palette.green NewOrg
                , viewOrgs sess.data model.orgEdit
                ]

        _ ->
            el [ padding 16 ] <| text "Sign in to view your orgs."


viewOrgs : Data.Data -> Maybe O.Org -> Element Msg
viewOrgs data orgEdit =
    column
        [ width fill
        , spacing 40
        ]
    <|
        List.map (\o -> viewOrg data.users o.mod o.org) <|
            mergeOrgEdit data.orgs orgEdit


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


viewOrg : List U.User -> Bool -> O.Org -> Element Msg
viewOrg users modded org =
    column
        ([ width fill
         , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
         , Border.color palette.black
         , spacing 6
         ]
            ++ (if modded then
                    [ Background.color palette.orange
                    , below <|
                        Button.viewRow
                            [ Button.view2 "discard" palette.pale <| DiscardOrgEdits
                            , Button.view2 "save" palette.green <| PostOrg org
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
        , el [ padding 16, Font.italic, Font.color palette.gray ] <| text "Users"
        , viewUsers org.users users
        ]


viewUsers : List O.UserRoles -> List U.User -> Element Msg
viewUsers userRoles users =
    column [ spacing 6, paddingEach { top = 0, right = 16, bottom = 0, left = 32 } ]
        (List.map
            (\ur ->
                case List.Extra.find (\u -> u.id == ur.userId) users of
                    Just user ->
                        el [ padding 16 ] <| text user.first

                    Nothing ->
                        el [ padding 16 ] <| text "User not found"
            )
            userRoles
        )



--viewItems : String -> O.Org -> Element Msg
--viewItems email org =
--    wrappedRow
--        [ width fill
--        , spacing 16
--        ]
--        [ viewUsers email org
--        , viewDevices org.devices
--        ]
--viewUsers : String -> O.Org -> Element Msg
--viewUsers email org =
--    column
--        []
--        [ Input.text
--            []
--            { onChange = EditEmail org.id
--            , text = email
--            , placeholder = Nothing
--            , label = label Input.labelAbove "Add user by email address"
--            }
--        , viewList "Users" viewUser org.users
--        ]
-- label : String -> Element Msg


label : (List (Attribute Msg) -> Element Msg -> Input.Label Msg) -> (String -> Input.Label Msg)
label kind =
    kind
        [ padding 16
        , Font.italic
        , Font.color palette.gray
        ]
        << text


viewDevices : List D.Device -> Element Msg
viewDevices =
    viewList "Devices" viewDevice


viewOrgName : String -> Element Msg
viewOrgName name =
    el
        [ padding 16
        , size.heading
        ]
    <|
        text name


viewList : String -> (a -> Element Msg) -> List a -> Element Msg
viewList name fn list =
    column
        [ alignTop
        , width (fill |> minimum 250)
        , spacing 16
        ]
    <|
        [ el [ padding 16 ] <| text name ]
            ++ List.map fn list


viewItem : List (Element Msg) -> Element Msg
viewItem =
    wrappedRow
        [ padding 16
        , spacing 25
        , Border.widthEach { top = 1, bottom = 0, left = 0, right = 0 }
        , Border.color palette.black
        , width fill
        ]


viewUser : U.User -> Element Msg
viewUser user =
    viewItem
        [ text user.first
        , text user.last
        ]



-- hasRole : String -> U.User -> Bool
--hasRole role user =
--    List.member role <| List.map .description user.roles
--viewRole :
--viewRole { role, value, action } =
--    Input.checkbox
--        [ padding 16 ]
--        { checked = value
--        , icon = Input.defaultCheckbox
--        , label = label Input.labelRight role
--        , onChange = action role
--        }


viewDevice : D.Device -> Element Msg
viewDevice device =
    viewItem
        [ text device.id
        , text device.config.description
        ]
