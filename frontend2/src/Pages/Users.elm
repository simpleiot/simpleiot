module Pages.Users exposing (Model, Msg, page)

import Dict exposing (Dict)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Generated.Params as Params
import Global
import Http
import Spa.Page
import Spa.Types as Types
import User as U
import Utils.Spa exposing (Page)
import Utils.Styles exposing (palette, size)


page : Page Params.Users Model Msg model msg appMsg
page =
    Spa.Page.component
        { title = always "Users"
        , init = always init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }



-- INIT


type alias Model =
    { userEdit : Maybe U.User
    }


init : Params.Users -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ =
    ( { userEdit = Nothing
      }
    , Cmd.none
    , Spa.Page.send Global.RequestUsers
    )



-- UPDATE


type Msg
    = PostUser U.User
    | EditUser U.User
    | DiscardUserEdits
    | NewUser


update : Types.PageContext route Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update context msg model =
    case msg of
        EditUser user ->
            ( { model | userEdit = Just user }
            , Cmd.none
            , Cmd.none
            )

        DiscardUserEdits ->
            ( { model | userEdit = Nothing }
            , Cmd.none
            , Cmd.none
            )

        PostUser user ->
            ( { model | userEdit = Nothing }
            , Cmd.none
            , case context.global of
                Global.SignedIn _ ->
                    Spa.Page.send <| Global.UpdateUser user

                Global.SignedOut _ ->
                    Cmd.none
            )

        NewUser ->
            ( { model | userEdit = Just U.empty }
            , Cmd.none
            , Cmd.none
            )



-- SUBSCRIPTIONS


subscriptions : Types.PageContext route Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.none



-- VIEW


view : Types.PageContext route Global.Model -> Model -> Element Msg
view context model =
    case context.global of
        Global.SignedIn sess ->
            column
                [ width fill, spacing 32 ]
                [ el [ padding 16, Font.size 24 ] <| text "Users"
                , el [ padding 16, width fill, Font.bold ] <| button "new user" palette.green NewUser
                , viewUsers sess.data.users model.userEdit
                ]

        _ ->
            el [ padding 16 ] <| text "Sign in to view users."


type alias UserMod =
    { user : U.User
    , mod : Bool
    }


mergeUserEdit : List U.User -> Maybe U.User -> List UserMod
mergeUserEdit users userEdit =
    case userEdit of
        Just edit ->
            let
                usersMapped =
                    List.map
                        (\u ->
                            let
                                _ =
                                    Debug.log "u" u
                            in
                            if edit.id == u.id then
                                { user = edit, mod = True }

                            else
                                { user = u, mod = False }
                        )
                        users
            in
            if edit.id == "" then
                [ { user = edit, mod = True } ] ++ usersMapped

            else
                usersMapped

        Nothing ->
            List.map (\u -> { user = u, mod = False }) users


viewUsers : List U.User -> Maybe U.User -> Element Msg
viewUsers users userEdit =
    let
        merged =
            mergeUserEdit users userEdit
    in
    column
        [ width fill
        , spacing 40
        ]
    <|
        List.map
            (\user ->
                viewUser user.mod user.user
            )
            merged


viewUser : Bool -> U.User -> Element Msg
viewUser modded user =
    wrappedRow
        ([ width fill
         , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
         , Border.color palette.black
         , spacing 6
         ]
            ++ (if modded then
                    [ Background.color palette.orange
                    , below <|
                        buttonRow
                            [ button "discard" palette.pale <| DiscardUserEdits
                            , button "save" palette.green <| PostUser user
                            ]
                    ]

                else
                    []
               )
        )
        [ viewTextProperty
            { name = "First name"
            , value = user.first
            , action = \x -> EditUser { user | first = x }
            }
        , viewTextProperty
            { name = "Last name"
            , value = user.last
            , action = \x -> EditUser { user | last = x }
            }
        , viewTextProperty
            { name = "Email"
            , value = user.email
            , action = \x -> EditUser { user | email = x }
            }
        , viewTextProperty
            { name = "Password"
            , value = user.pass
            , action = \x -> EditUser { user | pass = x }
            }
        ]


buttonRow : List (Element Msg) -> Element Msg
buttonRow =
    row
        [ Font.size 16
        , Font.bold
        , width fill
        , padding 16
        , spacing 16
        ]


button : String -> Color -> Msg -> Element Msg
button lbl color action =
    Input.button
        [ Background.color color
        , padding 16
        , width fill
        , Border.rounded 32
        ]
        { onPress = Just action
        , label = el [ centerX ] <| text lbl
        }


type alias TextProperty =
    { name : String
    , value : String
    , action : String -> Msg
    }


viewTextProperty : TextProperty -> Element Msg
viewTextProperty { name, value, action } =
    Input.text
        [ padding 16
        , width (fill |> minimum 150)
        , Border.width 0
        , Border.rounded 0
        , focused [ Background.color palette.yellow ]
        , Background.color palette.pale
        , spacing 0
        ]
        { onChange = action
        , text = value
        , placeholder = Nothing
        , label = label Input.labelAbove name
        }


label : (List (Attribute msg) -> Element msg -> Input.Label msg) -> (String -> Input.Label msg)
label kind =
    kind
        [ padding 16
        , Font.italic
        , Font.color palette.gray
        ]
        << text



--viewRoles =
--    row
--        []
--        << List.map viewRole
--viewRole { role, value, action } =
--    Input.checkbox
--        [ padding 16 ]
--        { checked = value
--        , icon = Input.defaultCheckbox
--        , label = label Input.labelRight role
--        , onChange = action
--        }
