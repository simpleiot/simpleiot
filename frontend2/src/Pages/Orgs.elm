module Pages.Orgs exposing (Model, Msg, page)

import Data.Data as Data
import Data.Device as D
import Data.Org as O
import Data.User as U
import Dict exposing (Dict)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Generated.Params as Params
import Global
import Spa.Page
import Spa.Types as Types
import Utils.Spa exposing (Page)
import Utils.Styles exposing (palette, size)


page : Page Params.Orgs Model Msg model msg appMsg
page =
    Spa.Page.component
        { title = always "Orgs"
        , init = init
        , update = always update
        , subscriptions = always subscriptions
        , view = view
        }



-- INIT


type alias Model =
    {}


init : Types.PageContext route Global.Model -> Params.Orgs -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ _ =
    ( empty
    , Cmd.none
    , Cmd.batch [ Spa.Page.send Global.RequestOrgs, Spa.Page.send Global.RequestUsers ]
    )


empty : Model
empty =
    {}



-- UPDATE


type Msg
    = EditEmail String String


update : Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update msg model =
    ( model, Cmd.none, Cmd.none )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.none



-- VIEW


view : Types.PageContext route Global.Model -> Model -> Element Msg
view context model =
    column
        [ width fill, spacing 32 ]
        [ el [ padding 16, Font.size 24 ] <| text "Orgs"
        , case context.global of
            Global.SignedIn sess ->
                viewOrgs sess.data

            _ ->
                el [ padding 16 ] <| text "Sign in to view your orgs."
        ]


viewOrgs : Data.Data -> Element Msg
viewOrgs data =
    column
        [ width fill
        , spacing 40
        ]
    <|
        List.map (viewOrg data.users) data.orgs


viewOrg : List U.User -> O.Org -> Element Msg
viewOrg users org =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color palette.black
        , spacing 6
        ]
        [ viewOrgName org.name

        --, viewItems (getEmail emails org.id) org
        ]



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
